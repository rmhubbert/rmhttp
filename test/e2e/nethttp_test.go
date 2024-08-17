package e2e

import (
	"net/http"
	"testing"

	"github.com/rmhubbert/rmhttp"
	"github.com/stretchr/testify/assert"
)

// the App to test
var netHTTPApp *rmhttp.NetHTTPApp = rmhttp.NewNetHTTP()

// A simple handlerFunc
var netHTTPHandler func(http.ResponseWriter, *http.Request) = func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("test body"))
}

// Test_Handler tests binding an rmhttp.Handler to  aspecific method & pattern
func Test_NetHTTPHandle(t *testing.T) {
	route := netHTTPApp.Handle("get", "/mypath", http.HandlerFunc(netHTTPHandler))
	assert.Equal(t, "GET /mypath", route.String())
}

// Test_HandlerFunc tests binding an rmhttp.HandlerFunc compatible function to
// a specific method & pattern
func Test_NetHTTPHandleFunc(t *testing.T) {
	route := netHTTPApp.HandleFunc("get", "/mypath", netHTTPHandler)
	assert.Equal(t, "GET /mypath", route.String())
}
