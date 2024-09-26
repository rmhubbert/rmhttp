package cors

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rmhubbert/rmhttp"
	"github.com/stretchr/testify/assert"
)

const (
	testAddress string = "localhost:8080"
)

func Test_Cors(t *testing.T) {
	testPattern := "/test"

	tests := []struct {
		name          string
		method        string
		requestMethod string
		expectedCode  int
		route         *rmhttp.Route
	}{
		{
			"CORS headers are returned with status 204 on OPTIONS requests",
			http.MethodOptions,
			http.MethodGet,
			http.StatusNoContent,
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
			"CORS headers are returned with status 200 on GET requests",
			http.MethodGet,
			http.MethodGet,
			http.StatusOK,
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
			test.route.Handler = Middleware()(test.route.Handler)

			url := fmt.Sprintf("http://%s%s", testAddress, testPattern)
			req, err := http.NewRequest(test.method, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			req.Header.Add("Access-Control-Request-Method", test.requestMethod)

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
		})
	}
}
