package rmhttp

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

// Test_HandleFunc checks that a handlerFunc can be successfully added to the App
func Test_HandleFunc(t *testing.T) {
	app := New()
	app.HandleFunc("get", "/handlefunc", createTestHandlerFunc(200, "test body"))

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

// Test_Convenience_Handlers checks that a handlerFunc can be successfully added to the App with
// any of the convenience methods.
func Test_Convenience_Handlers(t *testing.T) {
	app := New()
	tests := []struct {
		name    string
		method  string
		handler func(string, http.HandlerFunc) *Route
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
			test.handler(pattern, createTestHandlerFunc(200, "test body"))

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

// Test_Static checks that a static resource can be created and used by the App
func Test_Static(t *testing.T) {
	app := New()
	route := app.Static("/public", "./testdata")

	routes := app.Routes()
	assert.Len(t, routes, 1, "they should be equal")

	expectedKey := "GET /public"
	if route, ok := routes[expectedKey]; !ok {
		t.Errorf("route not found: %s", expectedKey)
	} else {
		assert.Equal(t, "GET", route.Method, "they should be equal")
		assert.Equal(t, "/public", route.Pattern, "they should be equal")
		assert.NotNil(t, route.Handler, "it should not be nil")
	}

	tests := []struct {
		name         string
		data         string
		pattern      string
		expectedCode int
	}{
		{"it serves a static file", "./testdata/test.html", "/public/test.html", http.StatusOK},
		{
			"it serves the index file with a trailing slash",
			"./testdata/index.html",
			"/public/",
			http.StatusOK,
		},
		{
			"it serves the index file without a trailing slash",
			"./testdata/index.html",
			"/public",
			http.StatusOK,
		},
		{
			"it redirects when the index file is called directly",
			"./testdata/index.html",
			"/public/index.html",
			http.StatusMovedPermanently,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Call the handler, test that the test data contents are returned.
			testData, err := os.ReadFile(test.data)
			if err != nil {
				t.Error("cannot read test data")
			}

			// Create a request that would trigger our test handler
			url := fmt.Sprintf("http://%s%s", testAddress, test.pattern)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			route.Handler.ServeHTTP(w, req)

			res := w.Result()
			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, test.expectedCode, res.StatusCode, "they should be equal")
			if res.StatusCode == http.StatusOK {
				assert.Contains(t, string(body), string(testData), "it should contain")
			}
		})
	}
}

// Test_Redirect checks that a redirect handler is successfully created added to the App
func Test_Redirect(t *testing.T) {
	app := New()
	pattern := "/redirect"
	tests := []struct {
		name         string
		target       string
		code         int
		expectedCode int
	}{
		{
			"Returns a redirect to http://localhost with a 301 response code",
			"http://localhost",
			http.StatusMovedPermanently,
			http.StatusMovedPermanently,
		},
		{
			"Returns a redirect to /other with a 301 response code",
			"/other",
			http.StatusMovedPermanently,
			http.StatusMovedPermanently,
		},
		{
			"Returns a redirect with a 307 response code when passed code is not between 300 - 308",
			"/other",
			http.StatusOK,
			http.StatusTemporaryRedirect,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			route := app.Redirect(pattern, test.target, test.code)

			routes := app.Routes()

			expectedKey := fmt.Sprintf("%s %s", "GET", pattern)
			if route, ok := routes[expectedKey]; !ok {
				t.Errorf("route not found: %s", expectedKey)
			} else {
				assert.Equal(t, "GET", route.Method, "they should be equal")
				assert.Equal(t, pattern, route.Pattern, "they should be equal")
				assert.NotNil(t, route.Handler, "it should not be nil")
			}

			// Create a request that would trigger our test handler
			url := fmt.Sprintf("http://%s%s", testAddress, pattern)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			route.Handler.ServeHTTP(w, req)

			res := w.Result()
			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

			assert.Equal(t, test.expectedCode, res.StatusCode, "they should be equal")
			assert.Equal(t, test.target, res.Header.Get("Location"), "they should be equal")
		})
	}
}

// Test_Routes checks that a list of current Routes is returned.
func Test_Routes(t *testing.T) {
	app := New()
	route := NewRoute(
		"GET",
		"/test",
		http.HandlerFunc(createTestHandlerFunc(http.StatusOK, "test body")),
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
		createTestHandlerFunc(http.StatusOK, testBody),
	)
	route.Use(
		createTestMiddlewareHandler("x-mw1", "mw1"),
		createTestMiddlewareHandler("x-mw2", "mw2"),
	)

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
	assert.Equal(
		t,
		fmt.Sprintf("%s %s", http.MethodGet, testPattern),
		pattern,
		"they should be the same",
	)

	// Call ServeHTTP on the handler so that we can inspect and confirm that the response status code
	// and body are what would expect to see from the test handler.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	res := w.Result()
	defer func() {
		err := res.Body.Close()
		if err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
	}()

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
