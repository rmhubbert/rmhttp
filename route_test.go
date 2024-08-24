package rmhttp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Route_Use(t *testing.T) {
	handler := createTestHandlerFunc(http.StatusOK, "test body", nil)
	m1 := createTestMiddlewareHandler("x-m1", "m1")
	m2 := createTestMiddlewareHandler("x-m2", "m2")

	route := NewRoute(http.MethodGet, "/route", HandlerFunc(handler))
	route.Use(m1, m2)

	assert.Len(t, route.middleware, 2, "they should be equal")
}
