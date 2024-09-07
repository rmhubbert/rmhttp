package rmhttp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"
)

// ------------------------------------------------------------------------------------------------
// SERVER CONFIG
// ------------------------------------------------------------------------------------------------

type ServerConfig struct {
	TimeoutConfig
	SSLConfig
	Host string
	Port int
	Cert string
	Key  string
}

// ------------------------------------------------------------------------------------------------
// SERVER
// ------------------------------------------------------------------------------------------------

// A Server wraps the standard library net/http.Server. It provide default lifecycle management
// and debugger logging on top of the expected http.Server behaviour.
type Server struct {
	Server              http.Server
	Router              http.Handler
	Logger              Logger
	Host                string
	Port                int
	cert                string
	key                 string
	writeTimeoutPadding time.Duration
}

// NewServer creates, initialises and returns a pointer to a Server.
func NewServer(
	config ServerConfig,
	router http.Handler,
	logger Logger,
) *Server {
	srv := Server{
		Server: http.Server{
			Handler:           router,
			ReadTimeout:       time.Duration(config.TCPReadTimeout) * time.Second,
			ReadHeaderTimeout: time.Duration(config.TCPReadHeaderTimeout) * time.Second,
			WriteTimeout:      time.Duration(config.TCPWriteTimeout) * time.Second,
			IdleTimeout:       time.Duration(config.TCPIdleTimeout) * time.Second,
		},
		Router:              router,
		Host:                config.Host,
		Port:                config.Port,
		cert:                config.Cert,
		key:                 config.Key,
		writeTimeoutPadding: time.Duration(config.TCPWriteTimeoutPadding) * time.Second,
		Logger:              logger,
	}
	srv.Server.Addr = srv.Address()
	return &srv
}

// maybeUpdateTimeout updates the http.Server read and write timeouts, if the passed duration
// is longer than the current values. We do this to ensure that the TCP connection does not
// timeout before the longest request timeout.
//
// See https://adam-p.ca/blog/2022/01/golang-http-server-timeouts/
func (srv *Server) maybeUpdateTimeout(timeout time.Duration) {
	readTimeout := timeout + srv.Server.ReadHeaderTimeout + srv.writeTimeoutPadding
	writeTimeout := timeout + srv.writeTimeoutPadding
	if readTimeout > srv.Server.ReadTimeout && writeTimeout > srv.Server.WriteTimeout {
		srv.Server.ReadTimeout = readTimeout
		srv.Server.WriteTimeout = writeTimeout
	}
}

// Address returns the server host and port as a formatted string ($HOST:$PORT).
func (srv *Server) Address() string {
	return fmt.Sprintf("%s:%d", srv.Host, srv.Port)
}

// ListenAndServe directly proxies the http.Server.ListenAndServe method. It starts the server
// without TLS support on the configured address and port.
func (srv *Server) ListenAndServe() error {
	return srv.Server.ListenAndServe()
}

// ListenAndServeTLS directly proxies the http.Server.ListenAndServeTLS method. It starts the
// server with TLS support on the configured address and port.
func (srv *Server) ListenAndServeTLS() error {
	return srv.Server.ListenAndServeTLS(srv.cert, srv.key)
}

// Shutdown directly proxies the net/http.Server.Shutdown method. It will stop the Server, if
// running.
func (srv *Server) Shutdown(ctx context.Context) error {
	return srv.Server.Shutdown(ctx)
}

// Start starts and manages the lifecycle of the underlyinh http.Server, facilitating graceful
// shutdowns and optional SSL support.
func (srv *Server) Start(useTLS bool) error {
	idleConnsClosed := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		// We received an interrupt signal, shut down.
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout:
			srv.Logger.Error(fmt.Sprintf("rmhttp server shutdown failed: %v", err))
		}
		close(idleConnsClosed)
	}()

	if useTLS && srv.cert != "" && srv.key != "" {
		srv.Logger.Info(fmt.Sprintf("starting rmhttp server with SSL on %v", srv.Address()))
		if err := srv.ListenAndServeTLS(); err != http.ErrServerClosed {
			srv.Logger.Error(fmt.Sprintf("rmhttp server with SSL start failed: %v", err))
			return err
		}
	} else {
		srv.Logger.Info(fmt.Sprintf("starting rmhttp server on %v", srv.Address()))
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			srv.Logger.Error(fmt.Sprintf("rmhttp server start failed: %v", err))
			return err
		}
	}

	<-idleConnsClosed
	return nil
}
