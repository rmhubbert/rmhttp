package rmhttp

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// A default config
var cfg, _ = LoadConfig(Config{})

// the Core to test
var app *App = New(cfg)

// A simple handlerFunc
var h HandlerFunc = HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("test body"))
	return nil
})

// Test_Handle checks that a handler can be successfully added to the App
func Test_Handle(t *testing.T) {
	app.Handle("get", "/handle", h)
	assert.Equal(t, 1, len(app.routes), "they should be equal")

	expectedKey := "GET /handle"
	handler, ok := app.routes[expectedKey]
	assert.Equal(t, true, ok, "they should be equal")
	assert.NotNil(t, h, handler, "it should not be nil")
}

// Test_HandleFunc checks that a handlerFunc can be successfully added to the App
func Test_HandleFunc(t *testing.T) {
	app.HandleFunc("get", "/handlefunc", h)
	expectedKey := "GET /handlefunc"

	handler, ok := app.routes[expectedKey]
	assert.Equal(t, true, ok, "they should be equal")
	assert.NotNil(t, h, handler, "it should not be nil")
}
