package rmhttp

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// ROUTER TESTS
// ------------------------------------------------------------------------------------------------

// Test_Router_ErrorHandlers checks that custom error handlers are used when internal 404 & 405
// errors are triggered.
func Test_Router_ErrorHandlers(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		method       string
		errorCode    int
		expectedCode int
		expectedBody string
		handler      http.HandlerFunc
	}{
		{
			"the custom 404 handler is used when an internal 404 error is thrown",
			"/notfound",
			http.MethodGet,
			http.StatusNotFound,
			http.StatusNotFound,
			"custom 404",
			http.HandlerFunc(createTestHandlerFunc(http.StatusNotFound, "custom 404")),
		},
		{
			"the custom 405 handler is used when an internal 405 error is thrown",
			"/pattern",
			http.MethodPost,
			http.StatusMethodNotAllowed,
			http.StatusMethodNotAllowed,
			"custom 405",
			http.HandlerFunc(createTestHandlerFunc(http.StatusMethodNotAllowed, "custom 405")),
		},
		{
			"the configured handler is used when no internal 404 or 405 error is thrown",
			"/pattern",
			http.MethodGet,
			http.StatusMethodNotAllowed,
			http.StatusOK,
			"pattern body",
			http.HandlerFunc(createTestHandlerFunc(http.StatusMethodNotAllowed, "custom 405")),
		},
	}

	out := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(out, nil))
	slog.SetDefault(logger)
	router := NewRouter()

	router.Handle(
		http.MethodGet,
		"/pattern",
		http.HandlerFunc(createTestHandlerFunc(http.StatusOK, "pattern body")),
	)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router.AddErrorHandler(test.errorCode, test.handler)

			// Create a request that would trigger our test handler
			url := fmt.Sprintf("http://%s%s", testAddress, test.pattern)
			req, err := http.NewRequest(test.method, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			// Call ServeHTTP on the handler so that we can inspect and confirm that
			// the response status code and body are what would expect to see from the
			// test handler.
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			res := w.Result()
			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, test.expectedBody, string(body), "they should be equal")
			assert.Equal(t, test.expectedCode, res.StatusCode, "they should be equal")
		})
	}
}
