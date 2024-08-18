package rmhttp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A simple handlerFunc
var h HandlerFunc = HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("test body"))
	return nil
})

// Test_Handle checks that a handler can be successfully added to the App
func Test_Handle(t *testing.T) {
	app := New()
	app.Handle("get", "/handle", h)
	assert.Equal(t, 1, len(app.routes), "they should be equal")

	expectedKey := "GET /handle"
	route, ok := app.routes[expectedKey]
	assert.Equal(t, true, ok, "they should be equal")
	assert.Equal(t, "GET", route.method, "they should be equal")
	assert.Equal(t, "/handle", route.pattern, "they should be equal")
	assert.NotNil(t, route.handler, "it should not be nil")
}

// Test_HandleFunc checks that a handlerFunc can be successfully added to the App
func Test_HandleFunc(t *testing.T) {
	app := New()
	app.HandleFunc("get", "/handlefunc", h)
	assert.Equal(t, 1, len(app.routes), "they should be equal")

	expectedKey := "GET /handlefunc"
	route, ok := app.routes[expectedKey]
	assert.Equal(t, true, ok, "they should be equal")
	assert.Equal(t, "GET", route.method, "they should be equal")
	assert.Equal(t, "/handlefunc", route.pattern, "they should be equal")
	assert.NotNil(t, route.handler, "it should not be nil")
}
