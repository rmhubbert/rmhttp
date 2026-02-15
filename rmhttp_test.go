package rmhttp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// RMHTTP TESTS
// ------------------------------------------------------------------------------------------------

// Test_Handle checks that a handler can be successfully added to the App
func Test_Handle(t *testing.T) {
	app := New()
	app.Handle("get", "/handle", http.HandlerFunc(createTestHandlerFunc(200, "test body")))
	routes := app.Routes()
	assert.Len(t, routes, 1, "they should be equal")

	expectedKey := "GET /handle"
	if route, ok := routes[expectedKey]; !ok {
		t.Errorf("route not found: %s", expectedKey)
	} else {
		assert.Equal(t, "GET", route.Method, "they should be equal")
		assert.Equal(t, "/handle", route.Pattern, "they should be equal")
		assert.NotNil(t, route.Handler, "it should not be nil")
	}
}

// Test_Pattern_Wildcard checks that wildcard patterns work correctly.
func Test_Pattern_Wildcard(t *testing.T) {
	app := New()

	app.Get("/files/{path...}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(r.PathValue("path")))
	})

	// Compile the app to load routes into the router
	app.Compile()

	// Test various wildcard scenarios
	tests := []struct {
		path       string
		expected   string
		statusCode int
	}{
		{"/files/a", "a", http.StatusOK},
		{"/files/a/b", "a/b", http.StatusOK},
		{"/files/a/b/c", "a/b/c", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			app.Router.ServeHTTP(w, req)
			assert.Equal(t, tt.statusCode, w.Code)
			assert.Equal(t, tt.expected, w.Body.String())
		})
	}
}

// Test_Pattern_MethodSpecific checks that method-specific patterns work correctly.
func Test_Pattern_MethodSpecific(t *testing.T) {
	app := New()

	app.Get("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("GET"))
	})

	app.Post("/users/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("POST"))
	})

	// Compile the app to load routes into the router
	app.Compile()

	// Test GET
	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	assert.Equal(t, "GET", w.Body.String())
	assert.Equal(t, http.StatusOK, w.Code)

	// Test POST
	req = httptest.NewRequest(http.MethodPost, "/users/123", nil)
	w = httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	assert.Equal(t, "POST", w.Body.String())
	assert.Equal(t, http.StatusOK, w.Code)
}

// Benchmark_Compile benchmarks the performance of compiling routes with middleware.
// It sets up an app with multiple routes and groups to simulate real-world usage.
func Benchmark_Compile(b *testing.B) {
	// Setup: create an app with a variety of routes and groups
	app := New()

	// Add direct routes
	for i := range 50 {
		app.Get(fmt.Sprintf("/route%d", i), createTestHandlerFunc(200, "ok"))
	}

	// Add groups with nested routes
	for i := range 10 {
		g := app.Group(fmt.Sprintf("/group%d", i))
		for j := range 10 {
			g.Get(fmt.Sprintf("/sub%d", j), createTestHandlerFunc(200, "ok"))
		}
		// Add some middleware to groups for realism
		g.Use(createTestMiddlewareHandler("x-group", fmt.Sprintf("group%d", i)))
	}

	// Benchmark Compile() - reset the router each iteration to avoid conflicts
	for b.Loop() {
		app.Router = NewRouter()
		app.Compile()
	}
}
