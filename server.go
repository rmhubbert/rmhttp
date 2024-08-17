package rmhttp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
)

// A Server wraps the standard library net/http.Server. It provide default lifecycle management
// and debugger logging on top of the expected http.Server behaviour.
type Server struct {
	Server  http.Server
	Router  http.Handler
	Logger  Logger
	Host    string
	Port    int
	SSLCert string
	SSLKey  string
}

// NewServer initialises and returns a pointer to a Server.
func NewServer(router http.Handler, host string, port int, SSLCert string, SSLKey string, logger Logger) *Server {
	srv := Server{
		Server:  http.Server{},
		Router:  router,
		Host:    host,
		Port:    port,
		SSLCert: SSLCert,
		SSLKey:  SSLKey,
		Logger:  logger,
	}
	srv.Server.Addr = srv.Address()
	srv.Server.Handler = router
	return &srv
}

func (srv *Server) Address() string {
	return fmt.Sprintf("%s:%d", srv.Host, srv.Port)
}

// ListenAndServe directly proxies the http.Server.ListenAndServe method. It starts the server without
// TLS support on the configured address and port.
func (srv *Server) ListenAndServe() error {
	return srv.Server.ListenAndServe()
}

// ListenAndServeTLS directly proxies the http.Server.ListenAndServeTLS method. It starts the server with
// TLS support on the configured address and port.
func (srv *Server) ListenAndServeTLS(cert string, key string) error {
	return srv.Server.ListenAndServeTLS(cert, key)
}

// Shutdown directly proxies the net/http.Server.Shutdown method. It will stop
// the Server, if running.
func (srv *Server) Shutdown(ctx context.Context) error {
	return srv.Server.Shutdown(ctx)
}

// handleLifeCycle
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

	if useTLS && srv.SSLCert != "" && srv.SSLKey != "" {
		srv.Logger.Info(fmt.Sprintf("starting rmhttp server with SSL on %v", srv.Address()))
		if err := srv.ListenAndServeTLS(srv.SSLCert, srv.SSLKey); err != http.ErrServerClosed {
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
