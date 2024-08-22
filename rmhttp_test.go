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
	route, ok := routes[expectedKey]
	assert.True(t, ok, "they should be equal")
	assert.Equal(t, "GET", route.Method(), "they should be equal")
	assert.Equal(t, "/handle", route.Pattern(), "they should be equal")
	assert.NotNil(t, route.Handler(), "it should not be nil")
}

// Test_HandleFunc checks that a handlerFunc can be successfully added to the App
func Test_HandleFunc(t *testing.T) {
	app := New()
	app.HandleFunc("get", "/handlefunc", createTestHandlerFunc(200, "test body", nil))

	routes := app.Routes()
	assert.Len(t, routes, 1, "they should be equal")

	expectedKey := "GET /handlefunc"
	route, ok := routes[expectedKey]
	assert.True(t, ok, "they should be equal")
	assert.Equal(t, "GET", route.Method(), "they should be equal")
	assert.Equal(t, "/handlefunc", route.Pattern(), "they should be equal")
	assert.NotNil(t, route.Handler(), "it should not be nil")
}

// Test_Routes checks that a list of current Routes is returned.
func Test_Routes(t *testing.T) {
	route := NewRoute(
		"GET",
		"/test",
		HandlerFunc(createTestHandlerFunc(http.StatusOK, "test body", nil)),
	)
	app := New()
	app.addRoute(route)

	routes := app.Routes()
	assert.Len(t, routes, 1, "they should be equal")

	r, ok := routes["GET /test"]
	if !ok {
		t.Error("route not found")
	}

	assert.Equal(t, "GET", r.Method(), "they should be equal")
	assert.Equal(t, "/test", r.Pattern(), "they should be equal")
}

// Test_Compile checks that the Routes can be compiled and loaded into the router's
// underlying http.ServeMux.
func Test_Compile(t *testing.T) {
	// Create the app
	app := New()
	// Create a handler to test with
	testPattern := "/test"
	testBody := "test body"
	app.HandleFunc(http.MethodGet, testPattern, createTestHandlerFunc(http.StatusOK, testBody, nil))
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

	// Lastly, call ServeHTTP on the handler so that we can inspect and confirm that
	// the response status code and body are what would expect to see from the
	// test handler.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("failed to read response body: %v", err)
	}
	assert.Equal(t, testBody, string(body), "they should be equal")
	assert.Equal(t, http.StatusOK, res.StatusCode, "they should be the same")
}
