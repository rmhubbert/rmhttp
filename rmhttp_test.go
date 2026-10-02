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
		//nolint:gosec
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

// serveCompiled sends one request through a freshly compiled App's router.
func serveCompiled(app *App, method, path string) *httptest.ResponseRecorder {
	app.Compile()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	return w
}

// Test_App_ErrorResponseModes pins the two error-response modes: an App with no error
// handlers (the default: stdlib bytes, pure mux serving) and a custom handler registered
// through the public API (interception active).
func Test_App_ErrorResponseModes(t *testing.T) {
	t.Run("the default produces the stdlib error bytes", func(t *testing.T) {
		app := New()
		app.Get("/exists", func(w http.ResponseWriter, _ *http.Request) {})

		notFound := serveCompiled(app, http.MethodGet, "/missing")
		assert.Equal(t, http.StatusNotFound, notFound.Code, "they should be equal")
		assert.Equal(t, "404 page not found\n", notFound.Body.String(), "they should be equal")
		assert.Equal(
			t,
			"text/plain; charset=utf-8",
			notFound.Header().Get("Content-Type"),
			"they should be equal",
		)

		app2 := New()
		app2.Get("/exists", func(w http.ResponseWriter, _ *http.Request) {})

		notAllowed := serveCompiled(app2, http.MethodPost, "/exists")
		assert.Equal(
			t,
			http.StatusMethodNotAllowed,
			notAllowed.Code,
			"they should be equal",
		)
		assert.Equal(t, "Method Not Allowed\n", notAllowed.Body.String(), "they should be equal")
		assert.Equal(t, "GET, HEAD", notAllowed.Header().Get("Allow"), "they should be equal")
	})

	t.Run("custom handlers intercept under the default config", func(t *testing.T) {
		app := New()
		app.Get("/exists", func(w http.ResponseWriter, _ *http.Request) {})
		app.StatusNotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("custom 404"))
		})

		w := serveCompiled(app, http.MethodGet, "/missing")

		assert.Equal(t, http.StatusNotFound, w.Code, "they should be equal")
		assert.Equal(t, "custom 404", w.Body.String(), "they should be equal")
	})
}
