package rmhttp

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// CONFIG TESTS
// ------------------------------------------------------------------------------------------------

var defaultTimeoutConfig = TimeoutConfig{
	TCPReadTimeout:         2,
	TCPReadHeaderTimeout:   1,
	TCPIdleTimeout:         120,
	TCPWriteTimeout:        5,
	TCPWriteTimeoutPadding: 1,
	RequestTimeout:         5,
	TimeoutMessage:         "Request Timeout",
}

var defaultSSLConfig = SSLConfig{
	Enable: false,
	Cert:   "",
	Key:    "",
}

var defaultConfig = Config{
	Host:    "",
	Port:    8080,
	Debug:   false,
	Logger:  nil,
	SSL:     defaultSSLConfig,
	Timeout: defaultTimeoutConfig,
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
		{"default host", cfg.Host, defaultConfig.Host},
		{"default port", cfg.Port, defaultConfig.Port},
		{"default debug flag", cfg.Debug, defaultConfig.Debug},
		{"default logger", cfg.Logger, defaultConfig.Logger},
		{"default SSL config", cfg.SSL, defaultConfig.SSL},
		{"default timeout config", cfg.Timeout, defaultConfig.Timeout},
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
	enablePanicRecovery := true
	enableHTTPErrorHandling := true
	enableHTTPLogging := true
	loggerAllowedMethods := []string{"GET", "POST"}

	// SSLConfig related env variables and config
	sslEnable := true
	sslCert := "/path/to/cert"
	sslKey := "/path/to/key"

	envSSLConfig := SSLConfig{
		Enable: sslEnable,
		Cert:   sslCert,
		Key:    sslKey,
	}

	// TimeoutConfig related env variables and config
	tcpReadTimeout := 10
	tcpReadHeaderTimeout := 10
	tcpIdleTimeout := 10
	tcpWriteTimeout := 10
	tcpWriteTimeoutBuffer := 10
	httpRequestTimeout := 10
	timeoutMessage := "Hello, World!"

	envTimeoutConfig := TimeoutConfig{
		TCPReadTimeout:         tcpReadTimeout,
		TCPReadHeaderTimeout:   tcpReadHeaderTimeout,
		TCPIdleTimeout:         tcpIdleTimeout,
		TCPWriteTimeout:        tcpWriteTimeout,
		TCPWriteTimeoutPadding: tcpWriteTimeoutBuffer,
		RequestTimeout:         httpRequestTimeout,
		TimeoutMessage:         timeoutMessage,
	}

	vars := map[string]string{
		"HOST":                       host,
		"PORT":                       strconv.Itoa(port),
		"DEBUG":                      strconv.FormatBool(debug),
		"ENABLE_PANIC_RECOVERY":      strconv.FormatBool(enablePanicRecovery),
		"ENABLE_HTTP_ERROR_HANDLING": strconv.FormatBool(enableHTTPErrorHandling),
		"ENABLE_HTTP_LOGGING":        strconv.FormatBool(enableHTTPLogging),
		"LOGGER_ALLOWED_METHODS":     strings.Join(loggerAllowedMethods, ","),
		"ENABLE_SSL":                 strconv.FormatBool(sslEnable),
		"SSL_CERT":                   sslCert,
		"SSL_KEY":                    sslKey,
		"TCP_READ_TIMEOUT":           strconv.Itoa(tcpReadTimeout),
		"TCP_READ_HEADER_TIMEOUT":    strconv.Itoa(tcpReadHeaderTimeout),
		"TCP_IDLE_TIMEOUT":           strconv.Itoa(tcpIdleTimeout),
		"TCP_WRITE_TIMEOUT":          strconv.Itoa(tcpWriteTimeout),
		"TCP_WRITE_TIMEOUT_BUFFER":   strconv.Itoa(tcpWriteTimeoutBuffer),
		"HTTP_REQUEST_TIMEOUT":       strconv.Itoa(httpRequestTimeout),
		"HTTP_TIMEOUT_MESSAGE":       timeoutMessage,
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
		{"host set from an environment variable", cfg.Host, host},
		{"port set from an environment variable", cfg.Port, port},
		{"debug flag set from an environment variable", cfg.Debug, debug},
		{"SSL config set from environment variables", cfg.SSL, envSSLConfig},
		{"timeout config set from environment variables", cfg.Timeout, envTimeoutConfig},
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

	// SSLConfig related env variables and config
	sslEnable := true
	sslCert := "/path/to/cert"
	sslKey := "/path/to/key"

	userSSLConfig := SSLConfig{
		Enable: sslEnable,
		Cert:   sslCert,
		Key:    sslKey,
	}

	// TimeoutConfig related env variables and config
	tcpReadTimeout := 10
	tcpReadHeaderTimeout := 10
	tcpIdleTimeout := 10
	tcpWriteTimeout := 10
	tcpWriteTimeoutBuffer := 10
	httpRequestTimeout := 10
	timeoutMessage := "Hello, World!"

	userTimeoutConfig := TimeoutConfig{
		TCPReadTimeout:         tcpReadTimeout,
		TCPReadHeaderTimeout:   tcpReadHeaderTimeout,
		TCPIdleTimeout:         tcpIdleTimeout,
		TCPWriteTimeout:        tcpWriteTimeout,
		TCPWriteTimeoutPadding: tcpWriteTimeoutBuffer,
		RequestTimeout:         httpRequestTimeout,
		TimeoutMessage:         timeoutMessage,
	}

	userConfig := Config{
		Host:    host,
		Port:    port,
		Debug:   debug,
		SSL:     userSSLConfig,
		Timeout: userTimeoutConfig,
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
		{"host set from a user defined config", cfg.Host, host},
		{"port set from a user defined config", cfg.Port, port},
		{"debug flag set from a user defined config", cfg.Debug, debug},
		{"SSL config set from a user defined config", cfg.SSL, userSSLConfig},
		{"timeout config set from a user defined config", cfg.Timeout, userTimeoutConfig},
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
	// SSLConfig related env variables and config
	// sslEnable := true
	// sslCert := "/path/to/cert"
	sslKey := "/path/to/key"

	partialSSLConfig := SSLConfig{
		// Enable: sslEnable,
		// Cert:   sslCert,
		Key: sslKey,
	}

	// TimeoutConfig related env variables and config
	tcpReadTimeout := 10
	// tcpReadHeaderTimeout := 10
	// tcpIdleTimeout := 10
	// tcpWriteTimeout := 10
	tcpWriteTimeoutBuffer := 10
	httpRequestTimeout := 10
	timeoutMessage := "Hello, World!"

	partialTimeoutConfig := TimeoutConfig{
		TCPReadTimeout: tcpReadTimeout,
		// TCPReadHeaderTimeout:  tcpReadHeaderTimeout,
		// TCPIdleTimeout:        tcpIdleTimeout,
		// TCPWriteTimeout:       tcpWriteTimeout,
		TCPWriteTimeoutPadding: tcpWriteTimeoutBuffer,
		RequestTimeout:         httpRequestTimeout,
		TimeoutMessage:         timeoutMessage,
	}

	partialConfig := Config{
		Host: host,
		// Port:                    port,
		// Debug:                   debug,
		SSL:     partialSSLConfig,
		Timeout: partialTimeoutConfig,
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
		{"host set from a partially defined user config", cfg.Host, host},
		{"port set from a partially defined user config", cfg.Port, defaultConfig.Port},
		{"debug flag set from a partially defined user config", cfg.Debug, defaultConfig.Debug},
		{
			"enable SSL flag set from a partially defined user config",
			cfg.SSL.Enable,
			defaultSSLConfig.Enable,
		},
		{
			"SSL certificate path set from a partially defined user config",
			cfg.SSL.Cert,
			defaultSSLConfig.Cert,
		},
		{
			"SSL key path set from a partially defined user config",
			cfg.SSL.Key,
			partialSSLConfig.Key,
		},
		{
			"TCP read timeout set from a partially defined user config",
			cfg.Timeout.TCPReadTimeout,
			partialTimeoutConfig.TCPReadTimeout,
		},
		{
			"TCP read header timeout set from a partially defined user config",
			cfg.Timeout.TCPReadHeaderTimeout,
			defaultTimeoutConfig.TCPReadHeaderTimeout,
		},
		{
			"TCP idle timeout set from a partially defined user config",
			cfg.Timeout.TCPIdleTimeout,
			defaultTimeoutConfig.TCPIdleTimeout,
		},
		{
			"TCP write timeout set from a partially defined user config",
			cfg.Timeout.TCPWriteTimeout,
			defaultTimeoutConfig.TCPWriteTimeout,
		},
		{
			"TCP write timeout buffer set from a partially defined user config",
			cfg.Timeout.TCPWriteTimeoutPadding,
			partialTimeoutConfig.TCPWriteTimeoutPadding,
		},
		{
			"HTTP request timeout set from a partially defined user config",
			cfg.Timeout.RequestTimeout,
			partialTimeoutConfig.RequestTimeout,
		},
		{
			"timeout message set from a partially defined user config",
			cfg.Timeout.TimeoutMessage,
			partialTimeoutConfig.TimeoutMessage,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			assert.Equal(t, test.Expected, test.Value, "they should be equal")
		})
	}
}
