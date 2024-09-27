package rmhttp

import (
	"fmt"
	"io"
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
	app.Handle("get", "/handle", HandlerFunc(createTestHandlerFunc(200, "test body", nil)))
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

// Test_HandleFunc checks that a handlerFunc can be successfully added to the App
func Test_HandleFunc(t *testing.T) {
	app := New()
	app.HandleFunc("get", "/handlefunc", createTestHandlerFunc(200, "test body", nil))

	routes := app.Routes()
	assert.Len(t, routes, 1, "they should be equal")

	expectedKey := "GET /handlefunc"
	if route, ok := routes[expectedKey]; !ok {
		t.Errorf("route not found: %s", expectedKey)
	} else {
		assert.Equal(t, "GET", route.Method, "they should be equal")
		assert.Equal(t, "/handlefunc", route.Pattern, "they should be equal")
		assert.NotNil(t, route.Handler, "it should not be nil")
	}
}

// Test_Get checks that a handlerFunc can be successfully added to the App with a GET method
func Test_Convenience_Handlers(t *testing.T) {
	app := New()
	tests := []struct {
		name    string
		method  string
		handler func(string, func(http.ResponseWriter, *http.Request) error) *Route
	}{
		{"Get creates and returns a Route with a GET method", "GET", app.Get},
		{"Post creates and returns a Route with a Post method", "POST", app.Post},
		{"Patch creates and returns a Route with a Patch method", "PATCH", app.Patch},
		{"Put creates and returns a Route with a Put method", "PUT", app.Put},
		{"Delete creates and returns a Route with a Delete method", "DELETE", app.Delete},
		{"Options creates and returns a Route with a Options method", "OPTIONS", app.Options},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pattern := "/handler"
			test.handler(pattern, createTestHandlerFunc(200, "test body", nil))

			routes := app.Routes()

			expectedKey := fmt.Sprintf("%s %s", test.method, pattern)
			if route, ok := routes[expectedKey]; !ok {
				t.Errorf("route not found: %s", expectedKey)
			} else {
				assert.Equal(t, test.method, route.Method, "they should be equal")
				assert.Equal(t, pattern, route.Pattern, "they should be equal")
				assert.NotNil(t, route.Handler, "it should not be nil")
			}
		})
	}
}

// Test_Routes checks that a list of current Routes is returned.
func Test_Routes(t *testing.T) {
	app := New()
	route := NewRoute(
		"GET",
		"/test",
		HandlerFunc(createTestHandlerFunc(http.StatusOK, "test body", nil)),
	)
	app.Route(route)

	routes := app.Routes()
	assert.Len(t, routes, 1, "they should be equal")

	expectedKey := "GET /test"
	if route, ok := routes[expectedKey]; !ok {
		t.Errorf("route not found: %s", expectedKey)
	} else {
		assert.Equal(t, "GET", route.Method, "they should be equal")
		assert.Equal(t, "/test", route.Pattern, "they should be equal")
	}
}

// Test_Compile checks that the Routes can be compiled and loaded into the router's
// underlying http.ServeMux.
func Test_Compile(t *testing.T) {
	// Create the app
	app := New()
	// Create a handler to test with
	testPattern := "/test"
	testBody := "test body"
	route := app.HandleFunc(
		http.MethodGet,
		testPattern,
		createTestHandlerFunc(http.StatusOK, testBody, nil),
	)
	route.Use(
		createTestMiddlewareHandler("x-mw1", "mw1"),
		createTestMiddlewareHandler("x-mw2", "mw2"),
	)

	// route.WithTimeout(10*time.Second, "TIMEOUT")
	// compile the routes
	app.Compile()

	// Create a request that would trigger our test handler
	url := fmt.Sprintf("http://%s%s", testAddress, testPattern)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	// Call Handler on the underlying http.ServeMux with the request we just created.
	// Assuming that Compile worked correctly and registered our test handler with
	// the mux, we should receive the handler back from this call.
	handler, pattern := app.Router.Mux.Handler(req)
	h := handler.(Handler)
	assert.Equal(
		t,
		fmt.Sprintf("%s %s", http.MethodGet, testPattern),
		pattern,
		"they should be the same",
	)

	// Call ServeHTTPWithError on the handler so that we can inspect and confirm that
	// the response status code and body are what would expect to see from the
	// test handler.
	w := httptest.NewRecorder()
	_ = h.ServeHTTPWithError(w, req)
	res := w.Result()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("failed to read response body: %v", err)
	}

	assert.Equal(t, http.StatusOK, res.StatusCode, "they should be equal")
	assert.Equal(t, testBody, string(body), "they should be equal")
	// Check that the middleware has been applied to the route. Our test middleware simply adds a
	// header.
	assert.Equal(t, "mw1", res.Header.Get("x-mw1"), "they should be equal")
	assert.Equal(t, "mw2", res.Header.Get("x-mw2"), "they should be equal")
}
