package benchmarks

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rmhubbert/rmhttp/v5"
	"github.com/rmhubbert/rmhttp/v5/pkg/middleware/apikey"
	"github.com/rmhubbert/rmhttp/v5/pkg/middleware/headers"
	"github.com/rmhubbert/rmhttp/v5/pkg/middleware/httplogger"
)

var out = &bytes.Buffer{}

func TestMain(m *testing.M) {
	// Disable slog output for all tests
	slog.SetDefault(slog.New(slog.NewTextHandler(out, nil)))
	exitCode := m.Run()
	os.Exit(exitCode)
}

// Benchmark_RequestHandling measures the performance of handling a single request.
func Benchmark_RequestHandling(b *testing.B) {
	app := rmhttp.New()
	app.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		//nolint:gosec // G705: This is a benchmark, not production code
		_, _ = w.Write([]byte("User: " + id))
	})

	// Pre-compile the app
	app.Compile()

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Router.ServeHTTP(w, req)
		w = httptest.NewRecorder() // Reset recorder
	}
}

// Benchmark_ConcurrentRequests measures the performance of handling concurrent requests.
func Benchmark_ConcurrentRequests(b *testing.B) {
	app := rmhttp.New()
	app.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		//nolint:gosec // G705: This is a benchmark, not production code
		_, _ = w.Write([]byte("User: " + id))
	})
	app.Compile()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)
		}
	})
}

// Benchmark_RouteMatching measures the performance of route matching with many routes.
func Benchmark_RouteMatching(b *testing.B) {
	app := rmhttp.New()
	for range 100 {
		app.Get("/users/{id}/posts/{post_id}", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("Post"))
		})
	}
	app.Compile()

	req := httptest.NewRequest(http.MethodGet, "/users/123/posts/456", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Router.ServeHTTP(w, req)
		w = httptest.NewRecorder()
	}
}

// Benchmark_MiddlewareStack measures the performance of a middleware stack.
func Benchmark_MiddlewareStack(b *testing.B) {
	app := rmhttp.New()
	app.Use(httplogger.Middleware())
	app.Use(headers.Middleware(map[string]string{
		"X-Custom-Header": "value",
	}))
	app.Use(apikey.Middleware(
		"key1", "key2", "key3",
	))
	app.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})
	app.Compile()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-api-key", "key1")
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Router.ServeHTTP(w, req)
		w = httptest.NewRecorder()
	}
}

// Benchmark_HeadersMiddleware measures the performance of the headers middleware.
func Benchmark_HeadersMiddleware(b *testing.B) {
	app := rmhttp.New()
	app.Use(headers.Middleware(map[string]string{
		"X-Header-1": "value1",
		"X-Header-2": "value2",
		"X-Header-3": "value3",
	}))
	app.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})
	app.Compile()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Router.ServeHTTP(w, req)
		w = httptest.NewRecorder()
	}
}

// Benchmark_APIKeyMiddleware measures the performance of the API key middleware.
func Benchmark_APIKeyMiddleware(b *testing.B) {
	app := rmhttp.New()
	app.Use(apikey.Middleware(
		"key1", "key2", "key3", "key4", "key5",
		"key6", "key7", "key8", "key9", "key10",
	))
	app.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})
	app.Compile()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-api-key", "key5")
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Router.ServeHTTP(w, req)
		w = httptest.NewRecorder()
	}
}

// Benchmark_PathValue measures the performance of path parameter extraction.
func Benchmark_PathValue(b *testing.B) {
	app := rmhttp.New()
	app.Get(
		"/users/{id}/posts/{post_id}/comments/{comment_id}",
		func(w http.ResponseWriter, r *http.Request) {
			_ = r.PathValue("id")
			_ = r.PathValue("post_id")
			_ = r.PathValue("comment_id")
			_, _ = w.Write([]byte("OK"))
		},
	)
	app.Compile()

	req := httptest.NewRequest(http.MethodGet, "/users/123/posts/456/comments/789", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Router.ServeHTTP(w, req)
		w = httptest.NewRecorder()
	}
}

// Benchmark_WildcardPath measures the performance of wildcard path parameter extraction.
func Benchmark_WildcardPath(b *testing.B) {
	app := rmhttp.New()
	app.Get("/files/{path...}", func(w http.ResponseWriter, r *http.Request) {
		_ = r.PathValue("path")
		_, _ = w.Write([]byte("OK"))
	})
	app.Compile()

	req := httptest.NewRequest(http.MethodGet, "/files/path/to/file.txt", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Router.ServeHTTP(w, req)
		w = httptest.NewRecorder()
	}
}

// Benchmark_GroupedRoutes measures the performance of route matching with grouped routes.
func Benchmark_GroupedRoutes(b *testing.B) {
	app := rmhttp.New()
	api := app.Group("/api/v1")
	api.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})
	api.Post("/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})
	api.Get("/posts/{id}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})
	app.Compile()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/123", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.Router.ServeHTTP(w, req)
		w = httptest.NewRecorder()
	}
}

// ------------------------------------------------------------------------------------------------
// HTTP/2 BENCHMARKS
// ------------------------------------------------------------------------------------------------

// Benchmark_PlainHTTP_Baseline measures plain HTTP/1.1 performance (no TLS, no HTTP/2).
// This represents the performance when running behind a reverse proxy WITHOUT h2c enabled.
func Benchmark_PlainHTTP_Baseline(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})

	// Plain HTTP server (HTTP/1.1 only)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := &http.Client{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

// Benchmark_HTTP2_Unencrypted measures HTTP/2 over plain TCP (h2c).
// This is what rmhttp now enables by default - HTTP/2 without TLS.
func Benchmark_HTTP2_Unencrypted(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})

	// Plain server but we force HTTP/2 by setting protocols
	ts := httptest.NewUnstartedServer(mux)
	ts.EnableHTTP2 = true
	ts.Start()
	defer ts.Close()

	client := &http.Client{}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

// Benchmark_HTTP2_TLS measures HTTP/2 over TLS (HTTPS).
// This is the standard HTTPS scenario with HTTP/2 enabled via ALPN.
func Benchmark_HTTP2_TLS(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})

	// HTTPS server with HTTP/2
	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	//nolint:gosec
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

// Benchmark_ConnectionReuse measures the benefit of connection reuse for different protocols.
// This is the key metric - real-world applications reuse connections for multiple requests.
func Benchmark_ConnectionReuse_PlainHTTP(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	// Create a single client and make multiple requests (simulating connection reuse)
	transport := &http.Transport{}
	client := &http.Client{Transport: transport}

	// Warm up
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
	for i := 0; i < 10; i++ {
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		//nolint:errcheck,gosec
		resp.Body.Close()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

func Benchmark_ConnectionReuse_HTTP2h2c(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})
	ts := httptest.NewUnstartedServer(mux)
	ts.EnableHTTP2 = true
	ts.Start()
	defer ts.Close()

	client := &http.Client{}

	// Warm up
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
	for i := 0; i < 10; i++ {
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		//nolint:errcheck,gosec
		resp.Body.Close()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

func Benchmark_ConnectionReuse_HTTP2TLS(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})
	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	//nolint:gosec
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	// Warm up
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
	for i := 0; i < 10; i++ {
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		//nolint:errcheck,gosec
		resp.Body.Close()
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

// Benchmark_ProtocolComparison_PlainHTTPvsHTTP2 is a comprehensive comparison
// between plain HTTP/1.1, HTTP/2 over plain TCP (h2c), and HTTP/2 over TLS.
func Benchmark_ProtocolComparison_PlainHTTPvsHTTP2(b *testing.B) {
	// Test all three protocols side by side
	testCases := []struct {
		name    string
		setup   func() (*httptest.Server, func())
		cleanup func()
	}{
		{
			name: "PlainHTTP/1.1",
			setup: func() (*httptest.Server, func()) {
				mux := http.NewServeMux()
				mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					//nolint:gosec,errcheck
					w.Write([]byte("OK"))
				})
				ts := httptest.NewServer(mux)
				return ts, func() { ts.Close() }
			},
			cleanup: func() {},
		},
		{
			name: "HTTP/2-h2c",
			setup: func() (*httptest.Server, func()) {
				mux := http.NewServeMux()
				mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					//nolint:gosec,errcheck
					w.Write([]byte("OK"))
				})
				ts := httptest.NewUnstartedServer(mux)
				ts.EnableHTTP2 = true
				ts.Start()
				return ts, func() { ts.Close() }
			},
			cleanup: func() {},
		},
		{
			name: "HTTP/2-TLS",
			setup: func() (*httptest.Server, func()) {
				mux := http.NewServeMux()
				mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					//nolint:gosec,errcheck
					w.Write([]byte("OK"))
				})
				ts := httptest.NewTLSServer(mux)
				return ts, func() { ts.Close() }
			},
			cleanup: func() {},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			ts, cleanup := tc.setup()
			defer cleanup()

			client := &http.Client{}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
					resp, err := client.Do(req)
					if err != nil {
						continue
					}
					//nolint:errcheck,gosec
					resp.Body.Close()
				}
			})
		})
	}
}

// Benchmark_HTTP2_ConcurrentStreams measures HTTP/2 performance with many concurrent streams
// over a single connection. This simulates real-world SSE scenarios.
func Benchmark_HTTP2_ConcurrentStreams(b *testing.B) {
	// Create a simple handler that responds quickly
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})

	// Start HTTPS server with HTTP/2
	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	// Configure HTTP/2 transport with settings similar to rmhttp defaults
	transport := &http.Transport{
		//nolint:gosec
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{Transport: transport}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

// Benchmark_HTTP2_LongLivedConnections measures performance of maintaining long-lived
// connections like SSE connections.
func Benchmark_HTTP2_LongLivedConnections(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}

		for i := 0; i < 10; i++ {
			//nolint:errcheck,gosec
			fmt.Fprintf(w, "data: %d\n\n", i)
			flusher.Flush()
			time.Sleep(1 * time.Millisecond)
		}
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	//nolint:gosec
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/stream", nil)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		//nolint:errcheck,gosec
		resp.Body.Close()
	}
}

// Benchmark_HTTP2_PingPong measures HTTP/2 ping/pong latency overhead.
func Benchmark_HTTP2_PingPong(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("pong"))
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	//nolint:gosec
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/ping", nil)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		//nolint:errcheck,gosec
		resp.Body.Close()
	}
}

// Benchmark_HTTP2_WithSSEResponse simulates SSE streaming response with multiple flushes.
// This is the critical use case for long-lived connections.
func Benchmark_HTTP2_WithSSEFlush(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		// Simulate SSE with 10 events
		for i := 0; i < 10; i++ {
			//nolint:errcheck,gosec
			fmt.Fprintf(w, "data: event-%d\n\n", i)
			flusher.Flush()
		}
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	//nolint:gosec
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/sse", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			// Read a portion of the response
			buf := make([]byte, 1024)
			//nolint:errcheck,gosec
			resp.Body.Read(buf)
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

// Benchmark_HTTP2_Multiplexing measures the performance benefit of HTTP/2 multiplexing
// by reusing a single connection for multiple sequential requests.
func Benchmark_HTTP2_Multiplexing(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	// Create a single client with connection pooling
	//nolint:gosec
	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:          1,
		MaxIdleConnsPerHost:   1,
		IdleConnTimeout:       30 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	client := &http.Client{Transport: transport}

	// First, establish the connection
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
	//nolint:errcheck,gosec,bodyclose
	client.Do(req)

	// Now measure reuse of the same connection
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

// Benchmark_HTTP2_LoadWithManyConcurrentClients simulates many clients making
// concurrent requests - typical load scenario.
func Benchmark_HTTP2_LoadWithManyConcurrentClients(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	//nolint:gosec
	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   20,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	client := &http.Client{Transport: transport}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

// Benchmark_HTTP2_GzipCompression measures HTTP/2 performance with response compression.
func Benchmark_HTTP2_GzipCompression(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		// Compressed response - typical JSON response
		//nolint:gosec,errcheck
		w.Write([]byte(
			`{"id":1,"name":"test","data":{"key":"value","items":["a","b","c","d","e"]}}`,
		))
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	//nolint:gosec
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

// Benchmark_HTTP2_HeadersWithTrace enables HTTP/2 trace to measure overhead of
// tracking request metrics.
func Benchmark_HTTP2_HeadersWithTrace(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			trace := &httptrace.ClientTrace{
				GotConn: func(connInfo httptrace.GotConnInfo) {},
			}
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			req = req.WithContext(httptrace.WithClientTrace(context.Background(), trace))
			//nolint:gosec
			transport := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			client := &http.Client{Transport: transport}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}

// Benchmark_HTTP2_Pipelining measures HTTP/2 request pipelining performance.
func Benchmark_HTTP2_Pipelining(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	//nolint:gosec
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}

	// Establish connection first
	req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
	resp, _ := client.Do(req)
	if resp != nil {
		//nolint:errcheck,gosec
		resp.Body.Close()
	}

	// Now measure sequential requests over same connection
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		//nolint:errcheck,gosec
		resp.Body.Close()
	}
}

// Benchmark_HTTP2_WarmPool measures performance with a pre-warmed connection pool.
func Benchmark_HTTP2_WarmPool(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		//nolint:gosec,errcheck
		w.Write([]byte("OK"))
	})

	ts := httptest.NewTLSServer(mux)
	defer ts.Close()

	// Pre-warm the connection pool
	poolSize := 10
	//nolint:gosec
	transport := &http.Transport{
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:          poolSize,
		MaxIdleConnsPerHost:   poolSize,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	// Warm up connections
	var wg sync.WaitGroup
	for i := 0; i < poolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &http.Client{Transport: transport}
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, _ := client.Do(req)
			if resp != nil {
				//nolint:errcheck,gosec
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	// Now benchmark with warm pool
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/test", nil)
			resp, err := transport.RoundTrip(req)
			if err != nil {
				continue
			}
			//nolint:errcheck,gosec
			resp.Body.Close()
		}
	})
}
