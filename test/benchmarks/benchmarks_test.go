package benchmarks

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

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
