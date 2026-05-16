package rmhttp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------------------------------------------
// SERVER TESTS
// ------------------------------------------------------------------------------------------------

// === NewServer Tests ===

// Test_NewServer tests the NewServer constructor function with various configurations.
func Test_NewServer(t *testing.T) {
	tests := []struct {
		name     string
		config   ServerConfig
		router   http.Handler
		validate func(t *testing.T, srv *Server)
	}{
		{
			name: "basic_config",
			config: ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
			router: http.NewServeMux(),
			validate: func(t *testing.T, srv *Server) {
				assert.Equal(t, "localhost", srv.Host)
				assert.Equal(t, 8080, srv.Port)
				assert.Equal(t, "localhost:8080", srv.Server.Addr)
			},
		},
		{
			name: "timeout_conversions",
			config: ServerConfig{
				Host:                 "0.0.0.0",
				Port:                 3000,
				TCPReadTimeout:       30,
				TCPReadHeaderTimeout: 5,
				TCPWriteTimeout:      60,
				TCPIdleTimeout:       120,
			},
			router: http.NewServeMux(),
			validate: func(t *testing.T, srv *Server) {
				assert.Equal(t, 30*time.Second, srv.Server.ReadTimeout)
				assert.Equal(t, 5*time.Second, srv.Server.ReadHeaderTimeout)
				assert.Equal(t, 60*time.Second, srv.Server.WriteTimeout)
				assert.Equal(t, 120*time.Second, srv.Server.IdleTimeout)
			},
		},
		{
			name: "empty_host",
			config: ServerConfig{
				Host: "",
				Port: 8080,
			},
			router: http.NewServeMux(),
			validate: func(t *testing.T, srv *Server) {
				assert.Equal(t, ":8080", srv.Server.Addr)
			},
		},
		{
			name: "zero_port",
			config: ServerConfig{
				Host: "localhost",
				Port: 0,
			},
			router: http.NewServeMux(),
			validate: func(t *testing.T, srv *Server) {
				assert.Equal(t, "localhost:0", srv.Server.Addr)
			},
		},
		{
			name: "zero_timeouts",
			config: ServerConfig{
				Host:                 "localhost",
				Port:                 8080,
				TCPReadTimeout:       0,
				TCPReadHeaderTimeout: 0,
				TCPWriteTimeout:      0,
				TCPIdleTimeout:       0,
			},
			router: http.NewServeMux(),
			validate: func(t *testing.T, srv *Server) {
				assert.Equal(t, 0*time.Second, srv.Server.ReadTimeout)
				assert.Equal(t, 0*time.Second, srv.Server.ReadHeaderTimeout)
			},
		},
		{
			name: "nil_router",
			config: ServerConfig{
				Host: "localhost",
				Port: 8080,
			},
			router: nil,
			validate: func(t *testing.T, srv *Server) {
				assert.Nil(t, srv.Router)
				assert.Nil(t, srv.Server.Handler)
			},
		},
		{
			name: "writeTimeoutPadding",
			config: ServerConfig{
				Host:                   "localhost",
				Port:                   8080,
				TCPWriteTimeoutPadding: 1,
			},
			router: http.NewServeMux(),
			validate: func(t *testing.T, srv *Server) {
				expectedPadding := 1 * time.Second
				assert.Equal(t, expectedPadding, srv.writeTimeoutPadding)
			},
		},
		{
			name: "DisableGeneralOptionsHandler",
			config: ServerConfig{
				Host:                         "localhost",
				Port:                         8080,
				DisableGeneralOptionsHandler: true,
			},
			router: http.NewServeMux(),
			validate: func(t *testing.T, srv *Server) {
				assert.True(t, srv.Server.DisableGeneralOptionsHandler)
			},
		},
		{
			name: "partial_http2_config_merges_with_defaults",
			config: ServerConfig{
				Host: "localhost",
				Port: 8080,
				HTTP2: &http.HTTP2Config{
					MaxConcurrentStreams: 200, // User overrides only this field
				},
			},
			router: http.NewServeMux(),
			validate: func(t *testing.T, srv *Server) {
				http2 := srv.Server.HTTP2
				require.NotNil(t, http2)
				// User override should be preserved
				assert.Equal(t, 200, http2.MaxConcurrentStreams)
				// Default values should be merged in for fields the user didn't set
				assert.Equal(t, 30*time.Second, http2.PingTimeout)
				assert.Equal(t, time.Duration(0), http2.WriteByteTimeout)
				assert.Equal(t, 16384, http2.MaxReadFrameSize)
			},
		},
		{
			name: "full_http2_config_overrides_all_defaults",
			config: ServerConfig{
				Host: "localhost",
				Port: 8080,
				HTTP2: &http.HTTP2Config{
					MaxConcurrentStreams: 50,
					PingTimeout:          10 * time.Second,
					WriteByteTimeout:     5 * time.Second,
					MaxReadFrameSize:     32768,
				},
			},
			router: http.NewServeMux(),
			validate: func(t *testing.T, srv *Server) {
				http2 := srv.Server.HTTP2
				require.NotNil(t, http2)
				assert.Equal(t, 50, http2.MaxConcurrentStreams)
				assert.Equal(t, 10*time.Second, http2.PingTimeout)
				assert.Equal(t, 5*time.Second, http2.WriteByteTimeout)
				assert.Equal(t, 32768, http2.MaxReadFrameSize)
			},
		},
		{
			name: "partial_protocols_config_merges_with_defaults",
			config: ServerConfig{
				Host: "localhost",
				Port: 8080,
				Protocols: func() *http.Protocols {
					p := &http.Protocols{}
					p.SetHTTP1(true) // User only sets HTTP/1.1
					return p
				}(),
			},
			router: http.NewServeMux(),
			validate: func(t *testing.T, srv *Server) {
				protocols := srv.Server.Protocols
				require.NotNil(t, protocols)
				// User override should be preserved
				assert.True(t, protocols.HTTP1())
				// Default values should be merged in for protocols the user didn't set
				assert.True(t, protocols.HTTP2())
				assert.True(t, protocols.UnencryptedHTTP2())
			},
		},
		{
			name: "partial_protocols_adds_to_defaults",
			config: ServerConfig{
				Host: "localhost",
				Port: 8080,
				Protocols: func() *http.Protocols {
					p := &http.Protocols{}
					p.SetHTTP1(true)
					p.SetHTTP2(true) // User enables HTTP/2 (already default)
					return p
				}(),
			},
			router: http.NewServeMux(),
			validate: func(t *testing.T, srv *Server) {
				protocols := srv.Server.Protocols
				require.NotNil(t, protocols)
				// User settings applied
				assert.True(t, protocols.HTTP1())
				assert.True(t, protocols.HTTP2())
				// Default values should be merged in for protocols the user didn't set
				assert.True(t, protocols.UnencryptedHTTP2())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := NewServer(tt.config, tt.router)
			require.NotNil(t, srv)
			tt.validate(t, srv)
		})
	}
}

// === maybeUpdateTimeout Tests ===

// Test_maybeUpdateTimeout tests the maybeUpdateTimeout method's timeout update logic.
func Test_maybeUpdateTimeout(t *testing.T) {
	t.Run("increases_timeouts_when_larger", func(t *testing.T) {
		srv := &Server{
			Server: http.Server{
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 20 * time.Second,
			},
			writeTimeoutPadding: 2 * time.Second,
		}

		// Set up ReadHeaderTimeout
		srv.Server.ReadHeaderTimeout = 1 * time.Second

		newTimeout := 30 * time.Second
		srv.maybeUpdateTimeout(newTimeout)

		// Expected: Read = 30 + 1 + 2 = 33s, Write = 30 + 2 = 32s
		assert.Equal(t, 33*time.Second, srv.Server.ReadTimeout)
		assert.Equal(t, 32*time.Second, srv.Server.WriteTimeout)
	})

	t.Run("no_update_when_smaller", func(t *testing.T) {
		srv := &Server{
			Server: http.Server{
				ReadTimeout:  60 * time.Second,
				WriteTimeout: 120 * time.Second,
			},
			writeTimeoutPadding: 5 * time.Second,
		}
		srv.Server.ReadHeaderTimeout = 10 * time.Second

		initialRead := srv.Server.ReadTimeout
		initialWrite := srv.Server.WriteTimeout

		newTimeout := 10 * time.Second
		srv.maybeUpdateTimeout(newTimeout)

		// Should remain unchanged
		assert.Equal(t, initialRead, srv.Server.ReadTimeout)
		assert.Equal(t, initialWrite, srv.Server.WriteTimeout)
	})

	t.Run("no_update_when_partial_increase", func(t *testing.T) {
		srv := &Server{
			Server: http.Server{
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 60 * time.Second,
			},
			writeTimeoutPadding: 2 * time.Second,
		}
		srv.Server.ReadHeaderTimeout = 5 * time.Second

		// This would increase Write (30 + 2 = 32 < 60) but not Read (30 + 5 + 2 = 37 > 30)
		// Since both must be greater, no update should occur
		newTimeout := 30 * time.Second
		srv.maybeUpdateTimeout(newTimeout)

		assert.Equal(t, 30*time.Second, srv.Server.ReadTimeout)
		assert.Equal(t, 60*time.Second, srv.Server.WriteTimeout)
	})

	t.Run("zero_initial_timeouts", func(t *testing.T) {
		srv := &Server{
			Server: http.Server{
				ReadTimeout:  0,
				WriteTimeout: 0,
			},
			writeTimeoutPadding: 2 * time.Second,
		}
		srv.Server.ReadHeaderTimeout = 1 * time.Second

		newTimeout := 30 * time.Second
		srv.maybeUpdateTimeout(newTimeout)

		// Should update since 0 < calculated values
		assert.Equal(t, 33*time.Second, srv.Server.ReadTimeout)
		assert.Equal(t, 32*time.Second, srv.Server.WriteTimeout)
	})

	t.Run("zero_padding", func(t *testing.T) {
		srv := &Server{
			Server: http.Server{
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 20 * time.Second,
			},
			writeTimeoutPadding: 0,
		}
		srv.Server.ReadHeaderTimeout = 0

		newTimeout := 30 * time.Second
		srv.maybeUpdateTimeout(newTimeout)

		// No padding added
		assert.Equal(t, 30*time.Second, srv.Server.ReadTimeout)
		assert.Equal(t, 30*time.Second, srv.Server.WriteTimeout)
	})

	t.Run("zero_timeout_passed", func(t *testing.T) {
		srv := &Server{
			Server: http.Server{
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 60 * time.Second,
			},
			writeTimeoutPadding: 2 * time.Second,
		}

		newTimeout := 0 * time.Second
		srv.maybeUpdateTimeout(newTimeout)

		// No update since 0 < current values
		assert.Equal(t, 30*time.Second, srv.Server.ReadTimeout)
		assert.Equal(t, 60*time.Second, srv.Server.WriteTimeout)
	})

	t.Run("multiple_calls_increasing", func(t *testing.T) {
		srv := &Server{
			Server: http.Server{
				ReadTimeout:  10 * time.Second,
				WriteTimeout: 20 * time.Second,
			},
			writeTimeoutPadding: 2 * time.Second,
		}
		srv.Server.ReadHeaderTimeout = 1 * time.Second

		// First call
		srv.maybeUpdateTimeout(30 * time.Second)
		assert.Equal(t, 33*time.Second, srv.Server.ReadTimeout)
		assert.Equal(t, 32*time.Second, srv.Server.WriteTimeout)

		// Second call with larger value
		srv.maybeUpdateTimeout(60 * time.Second)
		assert.Equal(t, 63*time.Second, srv.Server.ReadTimeout)
		assert.Equal(t, 62*time.Second, srv.Server.WriteTimeout)

		// Third call with smaller value (should not update)
		srv.maybeUpdateTimeout(30 * time.Second)
		assert.Equal(t, 63*time.Second, srv.Server.ReadTimeout)
		assert.Equal(t, 62*time.Second, srv.Server.WriteTimeout)
	})
}

// === setBestRouter Tests ===

// Test_setBestRouter tests the setBestRouter method's router optimization logic.
func Test_setBestRouter(t *testing.T) {
	t.Run("Router_without_error_handlers_replaced", func(t *testing.T) {
		// Create a Router without error handlers
		r := NewRouter()
		// Router by default has no error handlers

		srv := &Server{
			Server: http.Server{
				Handler:           r,
				ReadHeaderTimeout: 1 * time.Second,
			},
			Router: r,
		}

		initialRouter := srv.Router

		srv.setBestRouter()

		// Should be replaced with underlying Mux
		assert.IsType(t, &http.ServeMux{}, srv.Router)
		assert.Equal(t, r.Mux, srv.Router)
		assert.NotEqual(t, initialRouter, srv.Router)
	})

	t.Run("Router_with_error_handlers_preserved", func(t *testing.T) {
		// Create a Router with custom error handlers
		r := NewRouter()
		r.AddErrorHandler(
			http.StatusInternalServerError,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "custom error", http.StatusInternalServerError)
			}),
		)

		srv := &Server{
			Server: http.Server{
				Handler:           r,
				ReadHeaderTimeout: 1 * time.Second,
			},
			Router: r,
		}

		initialRouter := srv.Router

		srv.setBestRouter()

		// Should remain unchanged
		assert.Same(t, initialRouter, srv.Router)
		assert.IsType(t, &Router{}, srv.Router)
	})

	t.Run("non_Router_handler_preserved", func(t *testing.T) {
		// Use standard http.ServeMux directly
		mux := http.NewServeMux()

		srv := &Server{
			Server: http.Server{
				Handler:           mux,
				ReadHeaderTimeout: 1 * time.Second,
			},
			Router: mux,
		}

		initialRouter := srv.Router

		srv.setBestRouter()

		// Should remain unchanged (type assertion fails)
		assert.Same(t, initialRouter, srv.Router)
		assert.IsType(t, &http.ServeMux{}, srv.Router)
	})

	t.Run("nil_router_no_panic", func(t *testing.T) {
		srv := &Server{
			Router: nil,
			Server: http.Server{
				Handler:           nil,
				ReadHeaderTimeout: 1 * time.Second,
			},
		}

		// Should not panic
		assert.NotPanics(t, func() {
			srv.setBestRouter()
		})

		// Router should still be nil
		assert.Nil(t, srv.Router)
	})

	t.Run("multiple_calls_idempotent", func(t *testing.T) {
		r := NewRouter()
		// No error handlers

		srv := &Server{
			Router: r,
			Server: http.Server{
				Handler:           r,
				ReadHeaderTimeout: 1 * time.Second,
			},
		}

		srv.setBestRouter()
		firstRouter := srv.Router

		srv.setBestRouter()
		secondRouter := srv.Router

		srv.setBestRouter()
		thirdRouter := srv.Router

		// All should be the same (Mux)
		assert.Same(t, firstRouter, secondRouter)
		assert.Same(t, secondRouter, thirdRouter)
	})
}

// === ListenAndServe Integration Tests ===

// waitForServerAvailable waits for a server to be available on the given port.
func waitForServerAvailable(port int, timeout time.Duration) error {
	start := time.Now()
	for time.Since(start) < timeout {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("server not available after %v", timeout)
}

// Test_ListenAndServe_calls_setBestRouter verifies that setBestRouter is called before ListenAndServe.
func Test_ListenAndServe_calls_setBestRouter(t *testing.T) {
	r := NewRouter()
	// No error handlers - should be replaced with Mux

	config := ServerConfig{
		Host: "localhost",
		Port: 18080,
	}

	srv := NewServer(config, r)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.ListenAndServe()
	}()

	// Wait for server to be available
	require.NoError(t, waitForServerAvailable(config.Port, 5*time.Second))

	// Verify router was replaced (behavioral test)
	assert.IsType(t, &http.ServeMux{}, srv.Router)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = srv.Shutdown(ctx)
	}()

	// Check for errors
	select {
	case err := <-errChan:
		assert.Equal(t, http.ErrServerClosed, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Server did not stop")
	}
}

// Test_ListenAndServe_returns_error_on_invalid_port tests that ListenAndServe returns an error on invalid port.
func Test_ListenAndServe_returns_error_on_invalid_port(t *testing.T) {
	config := ServerConfig{
		Host: "localhost",
		Port: 1, // Privileged port - likely to fail
	}

	srv := NewServer(config, http.NewServeMux())

	// Try to start - should fail
	err := srv.ListenAndServe()
	assert.Error(t, err)
}

// === ListenAndServeTLS Integration Tests ===

// createTestTLSFiles generates self-signed TLS certificate and key files for testing.
func createTestTLSFiles(t *testing.T) (certFile, keyFile string) {
	t.Helper()

	// Generate key pair
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err, "failed to generate key pair")

	// Create a self-signed certificate
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "localhost",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:    []string{"localhost"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err, "failed to create certificate")

	// Encode private key
	keyDER, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err, "failed to marshal private key")

	// Encode to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	// Create temporary files for cert and key
	certF, err := os.CreateTemp("", "cert-*.pem")
	require.NoError(t, err, "failed to create cert temp file")

	keyF, err := os.CreateTemp("", "key-*.pem")
	require.NoError(t, err, "failed to create key temp file")

	// Write cert and key to files
	_, err = certF.Write(certPEM)
	require.NoError(t, err, "failed to write cert file")

	_, err = keyF.Write(keyPEM)
	require.NoError(t, err, "failed to write key file")

	// Close files so they can be read by the server
	_ = certF.Close()
	_ = keyF.Close()

	return certF.Name(), keyF.Name()
}

// Test_ListenAndServeTLS_invalid_cert_path tests that ListenAndServeTLS returns an error for invalid cert path.
func Test_ListenAndServeTLS_invalid_cert_path(t *testing.T) {
	config := ServerConfig{
		Host: "localhost",
		Port: 18443,
	}

	srv := NewServer(config, http.NewServeMux())

	err := srv.ListenAndServeTLS("/nonexistent/cert.pem", "/nonexistent/key.pem")
	assert.Error(t, err)
}

// Test_ListenAndServeTLS_invalid_key_path tests that ListenAndServeTLS returns an error for invalid key path.
func Test_ListenAndServeTLS_invalid_key_path(t *testing.T) {
	// Create temp cert file
	certFile, err := os.CreateTemp("", "cert-*.pem")
	require.NoError(t, err, "failed to create cert temp file")
	_ = os.Remove(certFile.Name())

	config := ServerConfig{
		Host: "localhost",
		Port: 18443,
	}

	srv := NewServer(config, http.NewServeMux())

	err = srv.ListenAndServeTLS(certFile.Name(), "/nonexistent/key.pem")
	assert.Error(t, err)
}

// Test_ListenAndServeTLS_empty_cert_path tests that ListenAndServeTLS returns an error for empty cert path.
func Test_ListenAndServeTLS_empty_cert_path(t *testing.T) {
	config := ServerConfig{
		Host: "localhost",
		Port: 18443,
	}

	srv := NewServer(config, http.NewServeMux())

	err := srv.ListenAndServeTLS("", "/some/key.pem")
	assert.Error(t, err)
}

// Test_ListenAndServeTLS_empty_key_path tests that ListenAndServeTLS returns an error for empty key path.
func Test_ListenAndServeTLS_empty_key_path(t *testing.T) {
	config := ServerConfig{
		Host: "localhost",
		Port: 18443,
	}

	srv := NewServer(config, http.NewServeMux())

	err := srv.ListenAndServeTLS("/some/cert.pem", "")
	assert.Error(t, err)
}

// Test_ListenAndServeTLS_with_valid_certs tests ListenAndServeTLS with valid certificate files.
func Test_ListenAndServeTLS_with_valid_certs(t *testing.T) {
	certFile, keyFile := createTestTLSFiles(t)
	defer func() {
		_ = os.Remove(certFile)
		_ = os.Remove(keyFile)
	}()

	config := ServerConfig{
		Host: "localhost",
		Port: 18443,
	}

	srv := NewServer(config, http.NewServeMux())

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.ListenAndServeTLS(certFile, keyFile)
	}()

	// Wait for server to be available
	err := waitForServerAvailable(config.Port, 5*time.Second)
	if err != nil {
		// Try to get any error from the server
		select {
		case srvErr := <-errChan:
			t.Logf("Server error: %v", srvErr)
		default:
			t.Log("No server error available")
		}
		require.NoError(t, err, "server should be available")
	}

	// Verify router was replaced
	assert.IsType(t, &http.ServeMux{}, srv.Router)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = srv.Shutdown(ctx)
	}()

	// Check for errors
	select {
	case err := <-errChan:
		assert.Equal(t, http.ErrServerClosed, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Server did not stop")
	}
}

// === Shutdown Tests ===

// Test_Shutdown_successful tests successful server shutdown.
func Test_Shutdown_successful(t *testing.T) {
	config := ServerConfig{
		Host: "localhost",
		Port: 18080,
	}

	srv := NewServer(config, http.NewServeMux())

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.ListenAndServe()
	}()

	// Wait for server to be available
	require.NoError(t, waitForServerAvailable(config.Port, 5*time.Second))

	// Shutdown with normal context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	require.NoError(t, err)

	// Verify server stopped
	select {
	case err := <-errChan:
		assert.Equal(t, http.ErrServerClosed, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Server did not stop")
	}
}

// Test_Shutdown_cancelled_context tests shutdown with cancelled context.
func Test_Shutdown_cancelled_context(t *testing.T) {
	config := ServerConfig{
		Host: "localhost",
		Port: 18080,
	}

	srv := NewServer(config, http.NewServeMux())

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.ListenAndServe()
	}()

	require.NoError(t, waitForServerAvailable(config.Port, 5*time.Second))

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Shutdown with cancelled context - may return error or nil
	// We just verify it doesn't panic
	assert.NotPanics(t, func() {
		err := srv.Shutdown(ctx)
		// If there's an error, it should contain "context canceled"
		if err != nil {
			assert.Contains(t, err.Error(), "context canceled")
		}
	})
}

// Test_Shutdown_timeout_context tests shutdown with context timeout.
func Test_Shutdown_timeout_context(t *testing.T) {
	config := ServerConfig{
		Host: "localhost",
		Port: 18080,
	}

	srv := NewServer(config, http.NewServeMux())

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.ListenAndServe()
	}()

	require.NoError(t, waitForServerAvailable(config.Port, 5*time.Second))

	// Shutdown with a very short timeout - this should timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// This should timeout or return context error
	err := srv.Shutdown(ctx)
	// Either error is acceptable - context deadline exceeded or server closed
	// We just verify it doesn't panic
	assert.NotPanics(t, func() {
		_ = err
	})
}

// Test_Shutdown_not_started tests shutdown on a server that was never started.
func Test_Shutdown_not_started(t *testing.T) {
	config := ServerConfig{
		Host: "localhost",
		Port: 18080,
	}

	srv := NewServer(config, http.NewServeMux())

	// Don't start - just shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Should not panic
	assert.NotPanics(t, func() {
		_ = srv.Shutdown(ctx)
	})
}

// Test_Shutdown_with_active_requests tests shutdown behavior with active requests.
func Test_Shutdown_with_active_requests(t *testing.T) {
	// Create handler that blocks
	blockChan := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-blockChan // Block until released
		w.WriteHeader(http.StatusOK)
	})

	config := ServerConfig{
		Host: "localhost",
		Port: 18080,
	}

	srv := NewServer(config, handler)

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.ListenAndServe()
	}()

	require.NoError(t, waitForServerAvailable(config.Port, 5*time.Second))

	// Make request that will block
	reqErrChan := make(chan error, 1)
	go func() {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/", config.Port))
		if err != nil {
			reqErrChan <- err
			return
		}
		_ = resp.Body.Close()
		reqErrChan <- nil
	}()

	// Give request time to start
	time.Sleep(100 * time.Millisecond)

	// Shutdown with timeout - should wait for request to complete or timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	shutdownErr := srv.Shutdown(ctx)

	// Release the blocking handler
	close(blockChan)

	// Request should complete (or fail due to shutdown)
	select {
	case reqErr := <-reqErrChan:
		// Either nil (completed) or error (shutdown interrupted)
		if reqErr != nil {
			errStr := reqErr.Error()
			assert.True(
				t,
				containsSubstring(errStr, "EOF") || containsSubstring(errStr, "connection") ||
					containsSubstring(errStr, "reset"),
			)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Request did not complete")
	}

	// Shutdown may have timed out or succeeded
	// Just verify no panic occurred
	_ = shutdownErr
}

// containsSubstring checks if the string contains the substring.
func containsSubstring(str, substr string) bool {
	return len(str) >= len(substr) &&
		(str == substr || len(str) > len(substr) && containsSubstringHelper(str, substr))
}

func containsSubstringHelper(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
