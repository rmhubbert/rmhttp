package rmhttp

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// CONFIG TESTS
// ------------------------------------------------------------------------------------------------

var defaultServerConfig = ServerConfig{
	TCPReadTimeout:         2,
	TCPReadHeaderTimeout:   1,
	TCPIdleTimeout:         120,
	TCPWriteTimeout:        5,
	TCPWriteTimeoutPadding: 1,
	RequestTimeout:         5,
	TimeoutMessage:         "Request Timeout",
	Port:                   8080,
}

var defaultConfig = Config{
	Debug:  false,
	Logger: nil,
	Server: defaultServerConfig,
}

// Test_LoadConfig_default tests the default config. It simulates no user config being passed
// and no related environment variables being set.
func Test_LoadConfig_default(t *testing.T) {
	cfg, err := LoadConfig(Config{})
	if err != nil {
		t.Errorf("LoadConfig returned error: %v", err)
	}

	tests := []struct {
		Name     string
		Value    any
		Expected any
	}{
		{"default debug flag", cfg.Debug, defaultConfig.Debug},
		{"default logger", cfg.Logger, defaultConfig.Logger},
		{"default timeout config", cfg.Server, defaultConfig.Server},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, test.Value, "they should be equal")
		})
	}
}

// Test_LoadConfig_from_env tests the default config, but with environment variables set. It simulates where the
// related environment variables have been set.
func Test_LoadConfig_from_env(t *testing.T) {
	// Config related env variables
	host := "localhost"
	port := 80
	debug := true

	// TimeoutConfig related env variables and config
	tcpReadTimeout := 10
	tcpReadHeaderTimeout := 10
	tcpIdleTimeout := 10
	tcpWriteTimeout := 10
	tcpWriteTimeoutPadding := 10
	httpRequestTimeout := 10
	timeoutMessage := "Hello, World!"

	envServerConfig := ServerConfig{
		TCPReadTimeout:         tcpReadTimeout,
		TCPReadHeaderTimeout:   tcpReadHeaderTimeout,
		TCPIdleTimeout:         tcpIdleTimeout,
		TCPWriteTimeout:        tcpWriteTimeout,
		TCPWriteTimeoutPadding: tcpWriteTimeoutPadding,
		RequestTimeout:         httpRequestTimeout,
		TimeoutMessage:         timeoutMessage,
		Host:                   host,
		Port:                   port,
	}

	vars := map[string]string{
		"HOST":                      host,
		"PORT":                      strconv.Itoa(port),
		"DEBUG":                     strconv.FormatBool(debug),
		"TCP_READ_TIMEOUT":          strconv.Itoa(tcpReadTimeout),
		"TCP_READ_HEADER_TIMEOUT":   strconv.Itoa(tcpReadHeaderTimeout),
		"TCP_IDLE_TIMEOUT":          strconv.Itoa(tcpIdleTimeout),
		"TCP_WRITE_TIMEOUT":         strconv.Itoa(tcpWriteTimeout),
		"TCP_WRITE_TIMEOUT_PADDING": strconv.Itoa(tcpWriteTimeoutPadding),
		"HTTP_REQUEST_TIMEOUT":      strconv.Itoa(httpRequestTimeout),
		"HTTP_TIMEOUT_MESSAGE":      timeoutMessage,
	}

	// Set the environment variables
	for key, value := range vars {
		err := os.Setenv(key, value)
		if err != nil {
			t.Errorf("failed to set environment variable: %v", err)
		}
	}

	cfg, err := LoadConfig(Config{})
	if err != nil {
		t.Errorf("LoadConfig returned error: %v", err)
	}

	tests := []struct {
		Name     string
		Value    any
		Expected any
	}{
		{"debug flag set from an environment variable", cfg.Debug, debug},
		{"timeout config set from environment variables", cfg.Server, envServerConfig},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, test.Value, "they should be equal")
		})
	}

	// Clean up the environment variables
	for key := range vars {
		err := os.Unsetenv(key)
		if err != nil {
			t.Errorf("failed to unset environment variable: %v", err)
		}
	}
}

// Test_LoadConfig_with_user_defined_config tests how the default config handles being merged with a user config.
// It simulates a complete user config being passed.
func Test_LoadConfig_with_user_defined_config(t *testing.T) {
	// Config related env variables
	host := "localhost"
	port := 80
	debug := true

	// ServerConfig related env variables and config
	tcpReadTimeout := 10
	tcpReadHeaderTimeout := 10
	tcpIdleTimeout := 10
	tcpWriteTimeout := 10
	tcpWriteTimeoutBuffer := 10
	httpRequestTimeout := 10
	timeoutMessage := "Hello, World!"

	userServerConfig := ServerConfig{
		TCPReadTimeout:         tcpReadTimeout,
		TCPReadHeaderTimeout:   tcpReadHeaderTimeout,
		TCPIdleTimeout:         tcpIdleTimeout,
		TCPWriteTimeout:        tcpWriteTimeout,
		TCPWriteTimeoutPadding: tcpWriteTimeoutBuffer,
		RequestTimeout:         httpRequestTimeout,
		TimeoutMessage:         timeoutMessage,
		Port:                   port,
		Host:                   host,
	}

	userConfig := Config{
		Debug:  debug,
		Server: userServerConfig,
	}

	cfg, err := LoadConfig(userConfig)
	if err != nil {
		t.Errorf("LoadConfig returned error: %v", err)
	}

	tests := []struct {
		Name     string
		Value    any
		Expected any
	}{
		{"debug flag set from a user defined config", cfg.Debug, debug},
		{"timeout config set from a user defined config", cfg.Server, userServerConfig},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, test.Value, "they should be equal")
		})
	}
}

// Test_LoadConfig_with_user_partially_defined_config tests how the default config handles being merged with a user config.
// It simulates a partially completed user config being passed.
func Test_LoadConfig_with_user_partially_defined_config(t *testing.T) {
	// Config related env variables
	host := "localhost"
	// port := 80
	// debug := true

	// ServerConfig related env variables and config
	tcpReadTimeout := 10
	// tcpReadHeaderTimeout := 10
	// tcpIdleTimeout := 10
	// tcpWriteTimeout := 10
	tcpWriteTimeoutBuffer := 10
	httpRequestTimeout := 10
	timeoutMessage := "Hello, World!"

	partialServerConfig := ServerConfig{
		TCPReadTimeout: tcpReadTimeout,
		// TCPReadHeaderTimeout:  tcpReadHeaderTimeout,
		// TCPIdleTimeout:        tcpIdleTimeout,
		// TCPWriteTimeout:       tcpWriteTimeout,
		TCPWriteTimeoutPadding: tcpWriteTimeoutBuffer,
		RequestTimeout:         httpRequestTimeout,
		TimeoutMessage:         timeoutMessage,
		Host:                   host,
	}

	partialConfig := Config{
		// Debug:                   debug,
		Server: partialServerConfig,
	}

	cfg, err := LoadConfig(partialConfig)
	if err != nil {
		t.Errorf("LoadConfig returned error: %v", err)
	}

	tests := []struct {
		Name     string
		Value    any
		Expected any
	}{
		{"debug flag set from a partially defined user config", cfg.Debug, defaultConfig.Debug},
		{
			"Host set from a partially defined user config",
			cfg.Server.Host,
			partialServerConfig.Host,
		},
		{
			"Port set from a partially defined user config",
			cfg.Server.Port,
			defaultServerConfig.Port,
		},
		{
			"TCP read timeout set from a partially defined user config",
			cfg.Server.TCPReadTimeout,
			partialServerConfig.TCPReadTimeout,
		},
		{
			"TCP read header timeout set from a partially defined user config",
			cfg.Server.TCPReadHeaderTimeout,
			defaultServerConfig.TCPReadHeaderTimeout,
		},
		{
			"TCP read timeout set from a partially defined user config",
			cfg.Server.TCPReadTimeout,
			partialServerConfig.TCPReadTimeout,
		},
		{
			"TCP read header timeout set from a partially defined user config",
			cfg.Server.TCPReadHeaderTimeout,
			defaultServerConfig.TCPReadHeaderTimeout,
		},
		{
			"TCP idle timeout set from a partially defined user config",
			cfg.Server.TCPIdleTimeout,
			defaultServerConfig.TCPIdleTimeout,
		},
		{
			"TCP write timeout set from a partially defined user config",
			cfg.Server.TCPWriteTimeout,
			defaultServerConfig.TCPWriteTimeout,
		},
		{
			"TCP write timeout buffer set from a partially defined user config",
			cfg.Server.TCPWriteTimeoutPadding,
			partialServerConfig.TCPWriteTimeoutPadding,
		},
		{
			"HTTP request timeout set from a partially defined user config",
			cfg.Server.RequestTimeout,
			partialServerConfig.RequestTimeout,
		},
		{
			"timeout message set from a partially defined user config",
			cfg.Server.TimeoutMessage,
			partialServerConfig.TimeoutMessage,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, test.Value, "they should be equal")
		})
	}
}
