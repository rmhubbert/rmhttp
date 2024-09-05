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

	route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), NewGroup(""))
	route.Use(m1, m2)

	assert.Len(t, route.Middleware, 2, "they should be equal")
}

// Test_Route_WithHeader checks that headers can be added to a Route.
func Test_Route_WithHeader(t *testing.T) {
	handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
	route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), NewGroup(""))

	route.WithHeader("x-h1", "h1")
	route.WithHeader("x-h2", "h2")

	assert.Len(t, route.Headers, 2, "they should be equal")
}

// Test_Route_WithTimeout checks that a timeout can be added to a Route.
func Test_Route_WithTimeout(t *testing.T) {
	handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
	route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), NewGroup(""))

	timeout := NewTimeout(5*time.Second, "Timeout!")
	route.WithTimeout(timeout.Duration, timeout.Message)
	to := Timeout(timeout)

	assert.IsType(t, to, route.Timeout, "they should be equal")
	assert.Equal(t, to.Duration, route.Timeout.Duration, "they should be equal")
	assert.Equal(t, to.Message, route.Timeout.Message, "they should be equal")
}
