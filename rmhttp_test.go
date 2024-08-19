package rmhttp

import (
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
	assert.Equal(t, 1, len(routes), "they should be equal")

	expectedKey := "GET /handle"
	route, ok := routes[expectedKey]
	assert.Equal(t, true, ok, "they should be equal")
	assert.Equal(t, "GET", route.Method(), "they should be equal")
	assert.Equal(t, "/handle", route.Pattern(), "they should be equal")
	assert.NotNil(t, route.Handler(), "it should not be nil")
}

// Test_HandleFunc checks that a handlerFunc can be successfully added to the App
func Test_HandleFunc(t *testing.T) {
	app := New()
	app.HandleFunc("get", "/handlefunc", createTestHandlerFunc(200, "test body", nil))

	routes := app.Routes()
	assert.Equal(t, 1, len(routes), "they should be equal")

	expectedKey := "GET /handlefunc"
	route, ok := routes[expectedKey]
	assert.Equal(t, true, ok, "they should be equal")
	assert.Equal(t, "GET", route.Method(), "they should be equal")
	assert.Equal(t, "/handlefunc", route.Pattern(), "they should be equal")
	assert.NotNil(t, route.Handler(), "it should not be nil")
}
