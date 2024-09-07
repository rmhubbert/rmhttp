package rmhttp

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test_Route_WithMiddleware checks that middleware can be added to a Route.
func Test_Route_WithMiddleware(t *testing.T) {
	handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
	m1 := createTestMiddlewareHandler("x-m1", "m1")
	m2 := createTestMiddlewareHandler("x-m2", "m2")

	route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), NewGroup(""))
	route.WithMiddleware(m1, m2)

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

	assert.Equal(t, timeout.Duration, route.Timeout.Duration, "they should be equal")
	assert.Equal(t, timeout.Message, route.Timeout.Message, "they should be equal")
	assert.True(t, timeout.Enabled, "they should be equal")
}

// Test_Route_ComputedTimeout checks that a parent Group timeout is used if a Timeout is not set
// directly on the Route.
func Test_Route_ComputedTimeout(t *testing.T) {
	t.Run("group timeout is used when route timeout has not been set", func(t *testing.T) {
		handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
		group := NewGroup("")
		group.WithTimeout(2, "group timeout")
		route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), group)

		assert.Equal(
			t,
			group.Timeout.Duration,
			route.ComputedTimeout().Duration,
			"they should be equal",
		)
		assert.Equal(
			t,
			group.Timeout.Message,
			route.ComputedTimeout().Message,
			"they should be equal",
		)
	})

	t.Run("timeout can be found with multiple groups", func(t *testing.T) {
		handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
		rootGroup := NewGroup("")
		rootGroup.WithTimeout(2, "root group timeout")
		group := NewGroup("/test")
		rootGroup.Group(group)
		route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), group)

		assert.Equal(
			t,
			rootGroup.Timeout.Duration,
			route.ComputedTimeout().Duration,
			"they should be equal",
		)
		assert.Equal(
			t,
			rootGroup.Timeout.Message,
			route.ComputedTimeout().Message,
			"they should be equal",
		)
	})
}

// Test_Route_ComputedPattern checks that the route pattern can be built with optional Group
// patterns prefixed.
func Test_Route_ComputedPattern(t *testing.T) {
	t.Run("route pattern is used when no group patterns are set", func(t *testing.T) {
		handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
		group := NewGroup("")
		route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), group)

		assert.Equal(t, "/route", route.ComputedPattern(), "they should be equal")
	})

	t.Run("group pattern is prepended to route pattern", func(t *testing.T) {
		handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
		group := NewGroup("/group")
		route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), group)

		assert.Equal(t, "/group/route", route.ComputedPattern(), "they should be equal")
	})

	t.Run("multiple group patterns are prepended to route pattern", func(t *testing.T) {
		handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
		rootGroup := NewGroup("/root")
		group := NewGroup("/group")
		rootGroup.Group(group)
		route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), group)

		assert.Equal(t, "/root/group/route", route.ComputedPattern(), "they should be equal")
	})
}

// Test_Route_ComputedHeaders checks that the correct headers are returned for a route,
// including any set on a parent Group
func Test_Route_ComputedHeaders(t *testing.T) {
	t.Run("route headers are used when no group headers are set", func(t *testing.T) {
		handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
		group := NewGroup("")
		route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), group)
		route.WithHeader("x-h1", "h1")
		route.WithHeader("x-h2", "h2")

		headers := route.ComputedHeaders()

		assert.Len(t, headers, 2, "they should be equal")
		assert.Equal(t, "h1", headers["x-h1"], "they should be equal")
		assert.Equal(t, "h2", headers["x-h2"], "they should be equal")
	})

	t.Run("group headers are added to route headers", func(t *testing.T) {
		handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
		group := NewGroup("")
		group.WithHeader("x-g1", "g1")
		group.WithHeader("x-g2", "g2")

		route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), group)
		route.WithHeader("x-h1", "h1")
		route.WithHeader("x-h2", "h2")

		headers := route.ComputedHeaders()

		assert.Len(t, headers, 4, "they should be equal")
		assert.Equal(t, "h1", headers["x-h1"], "they should be equal")
		assert.Equal(t, "h2", headers["x-h2"], "they should be equal")
		assert.Equal(t, "g1", headers["x-g1"], "they should be equal")
		assert.Equal(t, "g2", headers["x-g2"], "they should be equal")
	})

	t.Run("multiple group headers are added to route headers", func(t *testing.T) {
		handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
		rootGroup := NewGroup("/root")
		rootGroup.WithHeader("x-rg1", "rg1")
		rootGroup.WithHeader("x-rg2", "rg2")

		group := NewGroup("/group")
		group.WithHeader("x-g1", "g1")
		group.WithHeader("x-g2", "g2")

		rootGroup.Group(group)
		route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), group)
		route.WithHeader("x-h1", "h1")
		route.WithHeader("x-h2", "h2")

		headers := route.ComputedHeaders()

		assert.Len(t, headers, 6, "they should be equal")
		assert.Equal(t, "h1", headers["x-h1"], "they should be equal")
		assert.Equal(t, "h2", headers["x-h2"], "they should be equal")
		assert.Equal(t, "rg1", headers["x-rg1"], "they should be equal")
		assert.Equal(t, "rg2", headers["x-rg2"], "they should be equal")
		assert.Equal(t, "g1", headers["x-g1"], "they should be equal")
		assert.Equal(t, "g2", headers["x-g2"], "they should be equal")
	})

	t.Run("group headers do not overwrite route headers", func(t *testing.T) {
		handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
		group := NewGroup("")
		group.WithHeader("x-h1", "g1")

		route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler), group)
		route.WithHeader("x-h1", "h1")

		headers := route.ComputedHeaders()

		assert.Equal(t, "h1", headers["x-h1"], "they should be equal")
	})
}
