package e2e

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rmhubbert/rmhttp"
	"github.com/stretchr/testify/assert"
)

// Test_Handler tests binding an rmhttp.Handler to a method & pattern
func Test_Handle(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		pattern string
		path    string
		status  int
		body    string
		err     error
	}{
		{"GET the index", http.MethodGet, "/", "/", http.StatusOK, "get body", nil},
		{"POST to the index", http.MethodPost, "/", "/", http.StatusOK, "post body", nil},
		{"PUT to the index", http.MethodPut, "/", "/", http.StatusOK, "put body", nil},
		{"PATCH to the index", http.MethodPatch, "/", "/", http.StatusOK, "patch body", nil},
		{"DELETE to the index", http.MethodDelete, "/", "/", http.StatusOK, "delete body", nil},
		{"OPTIONS to the index", http.MethodOptions, "/", "/", http.StatusNoContent, "", nil},
		{"GET /test", http.MethodGet, "/test", "/test", http.StatusOK, "get body", nil},
		{"POST to /test", http.MethodPost, "/test", "/test", http.StatusOK, "post body", nil},
		{"PUT to /test", http.MethodPut, "/test", "/test", http.StatusOK, "put body", nil},
		{"PATCH to /test", http.MethodPatch, "/test", "/test", http.StatusOK, "patch body", nil},
		{"DELETE to /test", http.MethodDelete, "/test", "/test", http.StatusOK, "delete body", nil},
		{"OPTIONS to /test", http.MethodOptions, "/test", "/test", http.StatusNoContent, "", nil},
		{"GET /test/{id}", http.MethodGet, "/test/{id}", "/test/105", http.StatusOK, "get body", nil},
		{"POST to /test/{id}", http.MethodPost, "/test/{id}", "/test/105", http.StatusOK, "post body", nil},
		{"PUT to /test/{id}", http.MethodPut, "/test/{id}", "/test/105", http.StatusOK, "put body", nil},
		{"PATCH to /test/{id}", http.MethodPatch, "/test/{id}", "/test/105", http.StatusOK, "patch body", nil},
		{"DELETE to /test/{id}", http.MethodDelete, "/test/{id}", "/test/105", http.StatusOK, "delete body", nil},
		{"OPTIONS to /test/{id}", http.MethodOptions, "/test/{id}", "/test/105", http.StatusNoContent, "", nil},
	}

	app := rmhttp.New()
	defer app.Shutdown()
	for _, test := range tests {
		route := app.Handle(test.method, test.pattern, rmhttp.HandlerFunc(createHandlerFunc(test.status, test.body, test.err)))
		assert.Equal(t, fmt.Sprintf("%s %s", strings.ToUpper(test.method), strings.ToLower(test.pattern)), route.String())
	}
	startServer(app)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			url := fmt.Sprintf("http://%s%s", testAddress, test.path)
			req, err := http.NewRequest(test.method, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("get request failed: %v", err)
			}

			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, test.body, string(body), "they should be equal")
			assert.Equal(t, test.status, res.StatusCode, "they should be equal")
		})
	}
}

// Test_HandlerFunc tests binding an rmhttp.HandlerFunc compatible function to
// a specific method & pattern
func Test_HandleFunc(t *testing.T) {
	app := rmhttp.New()
	route := app.HandleFunc("get", "/handlefunc", createHandlerFunc(http.StatusOK, "test body", nil))
	assert.Equal(t, "GET /handlefunc", route.String())
}
