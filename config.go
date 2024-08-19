package rmhttp

import (
	"fmt"

	"dario.cat/mergo"
	"github.com/caarlos0/env/v11"
)

// ------------------------------------------------------------------------------------------------
// TIMEOUT CONFIG
// ------------------------------------------------------------------------------------------------
// The TimeoutConfig contains settings (with defaults) for configuring timeouts in the system.
// These settings mostly correlate to those used by the underlying http.Server
type TimeoutConfig struct {
	TCPReadTimeout        int    `env:"TCP_READ_TIMEOUT"         envDefault:"2"`
	TCPReadHeaderTimeout  int    `env:"TCP_READ_HEADER_TIMEOUT"  envDefault:"1"`
	TCPIdleTimeout        int    `env:"TCP_IDLE_TIMEOUT"         envDefault:"120"`
	TCPWriteTimeout       int    `env:"TCP_WRITE_TIMEOUT"        envDefault:"5"`
	TCPWriteTimeoutBuffer int    `env:"TCP_WRITE_TIMEOUT_BUFFER" envDefault:"1"`
	RequestTimeout        int    `env:"HTTP_REQUEST_TIMEOUT"     envDefault:"7"`
	TimeoutMessage        string `env:"HTTP_TIMEOUT_MESSAGE"     envDefault:"Request Timeout"`
}

// ------------------------------------------------------------------------------------------------
// SSL CONFIG
// ------------------------------------------------------------------------------------------------
// The SSLConfig contains settings (with defaults) for configuring SSL in the server.
type SSLConfig struct {
	Enable bool   `env:"ENABLE_SSL"`
	Cert   string `env:"SSL_CERT"`
	Key    string `env:"SSL_KEY"`
}

// ------------------------------------------------------------------------------------------------
// CORS CONFIG
// ------------------------------------------------------------------------------------------------
// The CorsConfig contains settings (with defaults) for configuring CORS in the router.
type CorsConfig struct {
	Enable               bool     `env:"ENABLE_CORS"                 envDefault:"false"`
	AllowedOrigin        string   `env:"CORS_ALLOWED_ORIGIN"         envDefault:"*"`
	AllowedMethods       []string `env:"CORS_ALLOWED_METHODS"        envDefault:"GET,POST,PUT,PATCH,DELETE"`
	AllowedHeaders       []string `env:"CORS_ALLOWED_HEADERS"        envDefault:"Origin,Authorization,X-Forwarded-For"`
	ExposedHeaders       []string `env:"CORS_EXPOSED_HEADERS"        envDefault:"Origin,Authorization,X-Forwarded-For"`
	MaxAge               int      `env:"CORS_MAX_AGE"                envDefault:"300"`
	OptionsSuccessStatus int      `env:"CORS_OPTIONS_SUCCESS_STATUS" envDefault:"204"`
	AllowCredentials     bool     `env:"CORS_ALLOW_CREDENTIALS"      envDefault:"false"`
	PreflightVary        []string `env:"CORS_PREFLIGHT_VARY"         envDefault:"Origin"`
}

// ------------------------------------------------------------------------------------------------
// CONFIG
// ------------------------------------------------------------------------------------------------
// The Config contains settings (with defaults) for configuring the app, server and router.
type Config struct {
	Host                    string   `env:"HOST"`
	Port                    int      `env:"PORT"                       envDefault:"8080"`
	Debug                   bool     `env:"DEBUG"`
	EnablePanicRecovery     bool     `env:"ENABLE_PANIC_RECOVERY"`
	EnableHTTPLogging       bool     `env:"ENABLE_HTTP_LOGGING"`
	EnableHTTPErrorHandling bool     `env:"ENABLE_HTTP_ERROR_HANDLING"`
	LoggerAllowedMethods    []string `env:"LOGGER_ALLOWED_METHODS"     envDefault:"GET,POST,PATCH,PUT,DELETE,OPTIONS"`
	Logger                  Logger
	Cors                    CorsConfig
	SSL                     SSLConfig
	Timeout                 TimeoutConfig
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

	// Merge the CORS config
	err = mergo.Merge(&config.Cors, cfg.Cors, mergo.WithOverride)
	if err != nil {
		return config, fmt.Errorf("failed to merge user supplied and default CORS configs: %v", err)
	}

	// Merge the SSL config
	err = mergo.Merge(&config.SSL, cfg.SSL, mergo.WithOverride)
	if err != nil {
		return config, fmt.Errorf("failed to merge user supplied and default SSL configs: %v", err)
	}

	// Merge the Timeout config
	err = mergo.Merge(&config.Timeout, cfg.Timeout, mergo.WithOverride)
	if err != nil {
		return config, fmt.Errorf(
			"failed to merge user supplied and default Timeout configs: %v",
			err,
		)
	}

	return config, nil
}
