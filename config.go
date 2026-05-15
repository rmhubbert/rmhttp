package rmhttp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"dario.cat/mergo"
	env "github.com/caarlos0/env/v11"
)

// ------------------------------------------------------------------------------------------------
// TIMEOUT CONFIG
// ------------------------------------------------------------------------------------------------

// defaultHTTP2Config returns the default HTTP/2 configuration for optimal performance.
// This enables HTTP/2 multiplexing and is tuned for high-concurrency workloads.
func defaultHTTP2Config() *http.HTTP2Config {
	return &http.HTTP2Config{
		MaxConcurrentStreams: 100,              // Default is typically 100 per connection
		PingTimeout:          30 * time.Second, // Keep connections alive for SSE
		WriteByteTimeout:     0,                // Don't timeout writes - critical for SSE streaming
		MaxReadFrameSize:     16384,            // Default 16KB - optimal for most workloads
	}
}

// defaultProtocols returns the default protocol configuration.
// This enables HTTP/1.1, HTTP/2 (over TLS), and HTTP/2 over plain TCP (h2c).
// h2c is particularly useful when running behind an SSL-terminating reverse proxy.
func defaultProtocols() *http.Protocols {
	p := &http.Protocols{}
	p.SetHTTP1(true)            // Enable HTTP/1.1
	p.SetHTTP2(true)            // Enable HTTP/2 (over TLS)
	p.SetUnencryptedHTTP2(true) // Enable h2c (HTTP/2 over plain TCP) - critical for reverse proxy
	return p
}

// The ServerConfig contains settings (with defaults) for configuring the underlying http.Server,
// as well as some additional timeout related properties. The server properties correlate to
// those found at https://pkg.go.dev/net/http#Server.
type ServerConfig struct {
	TCPReadTimeout               int    `env:"TCP_READ_TIMEOUT"          envDefault:"17"`
	TCPReadHeaderTimeout         int    `env:"TCP_READ_HEADER_TIMEOUT"   envDefault:"5"`
	TCPIdleTimeout               int    `env:"TCP_IDLE_TIMEOUT"          envDefault:"120"`
	TCPWriteTimeout              int    `env:"TCP_WRITE_TIMEOUT"         envDefault:"12"`
	TCPWriteTimeoutPadding       int    `env:"TCP_WRITE_TIMEOUT_PADDING" envDefault:"1"`
	RequestTimeout               int    `env:"HTTP_REQUEST_TIMEOUT"      envDefault:"10"`
	TimeoutMessage               string `env:"HTTP_TIMEOUT_MESSAGE"      envDefault:"Request Timeout"`
	MaxHeaderBytes               int    `env:"HTTP_MAX_HEADER_BYTES"`
	Host                         string `env:"HOST"`
	Port                         int    `env:"PORT"                      envDefault:"8080"`
	DisableGeneralOptionsHandler bool
	TLSConfig                    *tls.Config
	TLSNextProto                 map[string]func(*http.Server, *tls.Conn, http.Handler)
	ConnState                    func(net.Conn, http.ConnState)
	ErrorLog                     *log.Logger
	BaseContext                  func(net.Listener) context.Context
	ConnContext                  func(ctx context.Context, c net.Conn) context.Context
	HTTP2                        *http.HTTP2Config
	Protocols                    *http.Protocols

	// TCPKeepAlive enables TCP keep-alive on connections. This is particularly important
	// for long-lived connections like SSE, as it helps detect and close dead connections.
	// If set to false, HTTP keep-alives are also disabled.
	TCPKeepAlive bool `env:"TCP_KEEP_ALIVE" envDefault:"true"`
}

// ------------------------------------------------------------------------------------------------
// CONFIG
// ------------------------------------------------------------------------------------------------

// The Config contains settings (with defaults) for configuring the app, server and router.
type Config struct {
	Debug  bool `env:"DEBUG"`
	Server ServerConfig
}

// loadConfig parses the environment variables defined in the Config objects (with defaults), then merges those
// with the config that the user may have supplied. This function only gets called during app initialisation.
//
// This function will return a completed config, or error if the environment variables cannot be parsed.
func LoadConfig(cfg Config) (Config, error) {
	config := Config{}
	if err := env.Parse(&config); err != nil {
		return config, fmt.Errorf("failed to load environment variables: %v", err)
	}

	// Merge the main config
	err := mergo.Merge(&config, cfg, mergo.WithOverride)
	if err != nil {
		return config, fmt.Errorf("failed to merge user supplied and default configs: %v", err)
	}

	// Merge the Server config
	err = mergo.Merge(&config.Server, cfg.Server, mergo.WithOverride)
	if err != nil {
		return config, fmt.Errorf(
			"failed to merge user supplied and default server configs: %v",
			err,
		)
	}

	return config, nil
}
