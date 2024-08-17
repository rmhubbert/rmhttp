package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rmhubbert/rmhttp"
	"github.com/stretchr/testify/assert"
)

var (
	// A simple handlerFunc
	h func(http.ResponseWriter, *http.Request) error = func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test body"))
		return nil
	}
)

// Test_Handler tests binding an rmhttp.Handler to a method & pattern
func Test_Handle(t *testing.T) {
	route := app.Handle("GET", "/handle", rmhttp.HandlerFunc(h))
	assert.Equal(t, "GET /handle", route.String())

	url := fmt.Sprintf("http://%s/handle", testAddress)
	res, err := http.Get(url)
	if err != nil {
		t.Errorf("get request failed: %v", err)
	}

	assert.Equal(t, http.StatusOK, res.StatusCode, "they should be equal")
}

// Test_HandlerFunc tests binding an rmhttp.HandlerFunc compatible function to
// a specific method & pattern
func Test_HandleFunc(t *testing.T) {
	route := app.HandleFunc("get", "/handlefunc", h)
	assert.Equal(t, "GET /handlefunc", route.String())
}
