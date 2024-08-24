package rmhttp

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test_Route_Use checks that middleware can be added to a Route.
func Test_Route_Use(t *testing.T) {
	handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
	m1 := createTestMiddlewareHandler("x-m1", "m1")
	m2 := createTestMiddlewareHandler("x-m2", "m2")

	route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler))
	route.Use(m1, m2)

	assert.Len(t, route.middleware, 2, "they should be equal")
}

// Test_Route_WithHeader checks that headers can be added to a Route.
func Test_Route_WithHeader(t *testing.T) {
	handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
	route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler))

	route.WithHeader("x-h1", "h1")
	route.WithHeader("x-h2", "h2")

	assert.Len(t, route.headers, 2, "they should be equal")
}

// Test_Route_WithTimeout checks that a timeout can be added to a Route.
func Test_Route_WithTimeout(t *testing.T) {
	handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
	route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler))

	timeout := 5 * time.Second
	route.WithTimeout(timeout)
	to := Timeout(timeout)

	assert.IsType(t, to, route.timeout, "they should be equal")
	assert.Equal(t, to, route.timeout, "they should be equal")
}
