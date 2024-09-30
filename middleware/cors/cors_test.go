package cors

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rmhubbert/rmhttp"
	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// CORS TESTS
// ------------------------------------------------------------------------------------------------

const (
	testAddress string = "localhost:8080"
)

// Test_Cors checks that the expected CORS headers are returned for both preflight and standard
// requests. As this middleware wraps github.com/rs/cors, we only test the wrapper functionality
// and leave the in depth testing to the res/cors package itself.
func Test_Cors(t *testing.T) {
	testPattern := "/test"

	tests := []struct {
		name            string
		method          string
		requestMethod   string
		expectedCode    int
		expectedHeaders map[string]string
		options         Options
		route           *rmhttp.Route
	}{
		{
			"CORS preflight headers are returned with status 204 on OPTIONS requests with no config",
			http.MethodOptions,
			http.MethodGet,
			http.StatusNoContent,
			map[string]string{
				"Access-Control-Allow-Origin":  "*",
				"Access-Control-Allow-Methods": "GET",
			},
			Options{},
			rmhttp.NewRoute(
				http.MethodGet,
				testPattern,
				rmhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
					w.WriteHeader(http.StatusOK)
					return nil
				}),
			),
		},
		{
			"CORS standard headers are returned with status 200 on GET requests with no config",
			http.MethodGet,
			http.MethodGet,
			http.StatusOK,
			map[string]string{
				"Access-Control-Allow-Origin": "*",
			},
			Options{},
			rmhttp.NewRoute(
				http.MethodGet,
				testPattern,
				rmhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
					w.WriteHeader(http.StatusOK)
					return nil
				}),
			),
		},
		{
			"CORS preflight headers are returned with status 204 on OPTIONS requests with config",
			http.MethodOptions,
			http.MethodGet,
			http.StatusNoContent,
			map[string]string{
				"Access-Control-Allow-Origin":  "test.local",
				"Access-Control-Allow-Methods": "GET",
			},
			Options{
				AllowedOrigins: []string{"test.local"},
			},
			rmhttp.NewRoute(
				http.MethodGet,
				testPattern,
				rmhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
					w.WriteHeader(http.StatusOK)
					return nil
				}),
			),
		},
		{
			"CORS standard headers are returned with status 200 on GET requests with config",
			http.MethodGet,
			http.MethodGet,
			http.StatusOK,
			map[string]string{
				"Access-Control-Allow-Origin": "test.local",
			},
			Options{
				AllowedOrigins: []string{"test.local"},
			},
			rmhttp.NewRoute(
				http.MethodGet,
				testPattern,
				rmhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
					w.WriteHeader(http.StatusOK)
					return nil
				}),
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.route.Handler = Middleware(test.options)(test.route.Handler)

			url := fmt.Sprintf("http://%s%s", testAddress, testPattern)
			req, err := http.NewRequest(test.method, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			req.Header.Add("Access-Control-Request-Method", test.requestMethod)
			req.Header.Add("Origin", "test.local")

			w := httptest.NewRecorder()
			_ = test.route.Handler.ServeHTTPWithError(w, req)
			res := w.Result()
			defer res.Body.Close()

			assert.Equal(
				t,
				test.expectedCode,
				res.StatusCode,
				"they should be equal",
			)

			for k, v := range test.expectedHeaders {
				fmt.Println("HEADER: ", res.Header.Get(k))
				assert.Equal(t, v, res.Header.Get(k), "they should be equal")
			}
		})
	}
}
