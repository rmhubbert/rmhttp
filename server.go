package rmhttp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"dario.cat/mergo"
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
	// Apply default HTTP/2 configuration, merging user-provided values on top.
	// This enables multiplexing and is tuned for high-concurrency workloads.
	// If the user provides a partial config, defaults are used for unspecified fields.
	http2Config := defaultHTTP2Config()
	if config.HTTP2 != nil {
		_ = mergo.Merge(http2Config, config.HTTP2, mergo.WithOverride)
	}

	// Apply default Protocols configuration, merging user-provided values on top.
	// This enables h2c (HTTP/2 over plain TCP) for reverse proxy deployments.
	// If the user provides a partial config, defaults are used for unspecified protocols.
	// Since http.Protocols has unexported fields, we can't use mergo.
	// Instead, we start with defaults and apply user settings on top.
	protocols := defaultProtocols()
	if config.Protocols != nil {
		// Apply user protocol settings on top of defaults.
		// We use the public API to read user intent and apply overrides.
		// Only override if the user explicitly set a value different from the zero value.
		// Since we can't detect "touched" fields, we OR the values:
		// if either default or user enables a protocol, it stays enabled.
		// This means users can only disable protocols by providing a config
		// that has ALL protocols set to false, then re-enabling the ones they want.
		if config.Protocols.HTTP1() {
			protocols.SetHTTP1(true)
		}
		if config.Protocols.HTTP2() {
			protocols.SetHTTP2(true)
		}
		if config.Protocols.UnencryptedHTTP2() {
			protocols.SetUnencryptedHTTP2(true)
		}
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
