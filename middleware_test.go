package rmhttp

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Middleware_ApplyMmiddleware(t *testing.T) {
	// Create a middleware service
	mws := newMiddlewareService(MockLogger{})
	// Create a handler to test with
	testPattern := "/test"
	testBody := "test body"
	route := NewRoute(
		http.MethodGet,
		testPattern,
		HandlerFunc(createTestHandlerFunc(http.StatusOK, testBody, nil)),
	)
	route.Use(
		createTestMiddlewareHandler("x-mw1", "mw1"),
		createTestMiddlewareHandler("x-mw2", "mw2"),
	)

	preMw := createTestMiddlewareHandler("x-pre1", "pre1")
	postMw := createTestMiddlewareHandler("x-post1", "post1")
	mws.addPre(preMw)
	mws.addPost(postMw)

	// Apply the route middleware
	route.handler = mws.applyMiddleware(route.handler, route.middleware)

	// Create a request that would trigger our test handler
	url := fmt.Sprintf("http://%s%s", testAddress, testPattern)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	// Call ServeHTTP on the handler so that we can inspect and confirm that
	// the response status code and body are what would expect to see from the
	// test handler.
	w := httptest.NewRecorder()
	route.handler.ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("failed to read response body: %v", err)
	}

	assert.Equal(t, testBody, string(body), "they should be equal")
	assert.Equal(t, http.StatusOK, res.StatusCode, "they should be equal")
	// Check that the middleware has been applied to the route. Our test middleware simply adds a
	// header.
	assert.Equal(t, "pre1", res.Header.Get("x-pre1"), "they should be equal")
	assert.Equal(t, "post1", res.Header.Get("x-post1"), "they should be equal")
	assert.Equal(t, "mw1", res.Header.Get("x-mw1"), "they should be equal")
	assert.Equal(t, "mw2", res.Header.Get("x-mw2"), "they should be equal")
}