package rmhttp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ------------------------------------------------------------------------------------------------
// SERVER
// ------------------------------------------------------------------------------------------------

// A Server wraps the standard library net/http.Server. It provide default lifecycle management
// and debugger logging on top of the expected http.Server behaviour.
type Server struct {
	Server              http.Server
	Router              http.Handler
	Logger              *slog.Logger
	Port                int
	Host                string
	writeTimeoutPadding time.Duration
}

// NewServer creates, initialises and returns a pointer to a Server.
func NewServer(
	config ServerConfig,
	router http.Handler,
	logger *slog.Logger,
) *Server {
	srv := Server{
		Server: http.Server{
			Handler:                      router,
			ReadTimeout:                  time.Duration(config.TCPReadTimeout) * time.Second,
			ReadHeaderTimeout:            time.Duration(config.TCPReadHeaderTimeout) * time.Second,
			WriteTimeout:                 time.Duration(config.TCPWriteTimeout) * time.Second,
			IdleTimeout:                  time.Duration(config.TCPIdleTimeout) * time.Second,
			DisableGeneralOptionsHandler: config.DisableGeneralOptionsHandler,
			// TLSConfig:                    config.TLSConfig,
			// MaxHeaderBytes:               config.MaxHeaderBytes,
			// TLSNextProto:                 config.TLSNextProto,
			// ConnState:                    config.ConnState,
			// ErrorLog:                     config.ErrorLog,
			// BaseContext:                  config.BaseContext,
			// ConnContext:                  config.ConnContext,
			// HTTP2:                        config.HTTP2,
			// Protocols:                    config.Protocols,
		},
		Router:              router,
		Host:                config.Host,
		Port:                config.Port,
		writeTimeoutPadding: time.Duration(config.TCPWriteTimeoutPadding) * time.Second,
		Logger:              logger,
	}
	srv.Server.Addr = fmt.Sprintf("%s:%d", config.Host, config.Port)
	fmt.Printf("Server listening on %s\n", srv.Server.Addr)
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

// ListenAndServe directly proxies the http.Server.ListenAndServe method. It starts the server
// without TLS support on the configured address and port.
func (srv *Server) ListenAndServe() error {
	return srv.Server.ListenAndServe()
}

// ListenAndServeTLS directly proxies the http.Server.ListenAndServeTLS method. It starts the
// server with TLS support on the configured address and port.
func (srv *Server) ListenAndServeTLS(cert string, key string) error {
	return srv.Server.ListenAndServeTLS(cert, key)
}

// Shutdown directly proxies the net/http.Server.Shutdown method. It will stop the Server, if
// running.
func (srv *Server) Shutdown(ctx context.Context) error {
	return srv.Server.Shutdown(ctx)
}
