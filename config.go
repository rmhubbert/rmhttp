package rmhttp

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"

	"dario.cat/mergo"
	env "github.com/caarlos0/env/v11"
)

// ------------------------------------------------------------------------------------------------
// TIMEOUT CONFIG
// ------------------------------------------------------------------------------------------------

// The ServerConfig contains settings (with defaults) for configuring the underlying http.Server,
// as well as some additional timeout related properties. The server properties correlate to
// those found at https://pkg.go.dev/net/http#Server.
type ServerConfig struct {
	TCPReadTimeout               int    `env:"TCP_READ_TIMEOUT"          envDefault:"2"`
	TCPReadHeaderTimeout         int    `env:"TCP_READ_HEADER_TIMEOUT"   envDefault:"1"`
	TCPIdleTimeout               int    `env:"TCP_IDLE_TIMEOUT"          envDefault:"120"`
	TCPWriteTimeout              int    `env:"TCP_WRITE_TIMEOUT"         envDefault:"5"`
	TCPWriteTimeoutPadding       int    `env:"TCP_WRITE_TIMEOUT_PADDING" envDefault:"1"`
	RequestTimeout               int    `env:"HTTP_REQUEST_TIMEOUT"      envDefault:"5"`
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
}

// ------------------------------------------------------------------------------------------------
// CONFIG
// ------------------------------------------------------------------------------------------------

// The Config contains settings (with defaults) for configuring the app, server and router.
type Config struct {
	Debug  bool `env:"DEBUG"`
	Logger *slog.Logger
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
