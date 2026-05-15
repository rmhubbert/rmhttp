package rmhttp

import (
	"context"
	"fmt"
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
	Port                int
	Host                string
	writeTimeoutPadding time.Duration
}

// NewServer creates, initialises and returns a pointer to a Server.
func NewServer(
	config ServerConfig,
	router http.Handler,
) *Server {
	// Apply default HTTP/2 configuration if not provided
	// This enables multiplexing and is tuned for high-concurrency workloads
	http2Config := config.HTTP2
	if http2Config == nil {
		http2Config = defaultHTTP2Config()
	}

	// Apply default Protocols configuration if not provided
	// This enables h2c (HTTP/2 over plain TCP) for reverse proxy deployments
	protocols := config.Protocols
	if protocols == nil {
		protocols = defaultProtocols()
	}

	srv := Server{
		Server: http.Server{
			Handler:                      router,
			ReadTimeout:                  time.Duration(config.TCPReadTimeout) * time.Second,
			ReadHeaderTimeout:            time.Duration(config.TCPReadHeaderTimeout) * time.Second,
			WriteTimeout:                 time.Duration(config.TCPWriteTimeout) * time.Second,
			IdleTimeout:                  time.Duration(config.TCPIdleTimeout) * time.Second,
			DisableGeneralOptionsHandler: config.DisableGeneralOptionsHandler,
			HTTP2:                        http2Config,
			Protocols:                    protocols,
		},
		Router:              router,
		Host:                config.Host,
		Port:                config.Port,
		writeTimeoutPadding: time.Duration(config.TCPWriteTimeoutPadding) * time.Second,
	}
	srv.Server.Addr = fmt.Sprintf("%s:%d", config.Host, config.Port)

	// Configure HTTP keep-alive settings
	// Note: TCP-level keep-alive (for detecting dead connections in long-lived SSE)
	// requires a custom listener. These settings control HTTP keep-alive behavior.
	if !config.TCPKeepAlive {
		srv.Server.SetKeepAlivesEnabled(false)
	}

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

// bestRouter returns the faster router for the Server, given the current configuration. If the
// router has custom error handlers, it returns the Router itself; otherwise, it returns the
// Router's underlying Mux.
func (srv *Server) setBestRouter() {
	if r, ok := srv.Router.(*Router); ok {
		if !r.HasErrorHandlers() {
			srv.Router = r.Mux
		}
	}
}

// ListenAndServe directly proxies the http.Server.ListenAndServe method. It starts the server
// without TLS support on the configured address and port.
func (srv *Server) ListenAndServe() error {
	srv.setBestRouter()
	return srv.Server.ListenAndServe()
}

// ListenAndServeTLS directly proxies the http.Server.ListenAndServeTLS method. It starts the
// server with TLS support on the configured address and port.
func (srv *Server) ListenAndServeTLS(cert string, key string) error {
	srv.setBestRouter()
	return srv.Server.ListenAndServeTLS(cert, key)
}

// Shutdown directly proxies the net/http.Server.Shutdown method. It will stop the Server, if
// running.
func (srv *Server) Shutdown(ctx context.Context) error {
	return srv.Server.Shutdown(ctx)
}
