package headers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// HEADERS TESTS
// ------------------------------------------------------------------------------------------------

// Test_Headers checks that headers are properly added to the response.
func Test_Headers(t *testing.T) {
	tests := []struct {
		name          string
		expectedKey   string
		expectedValue string
		handler       http.Handler
	}{
		{
			"Headers are added to response",
			"x-key",
			"value",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := w.Write([]byte("Hello"))
				if err != nil {
					return
				}
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			headers := map[string]string{
				"x-key": "value",
			}
			h := Middleware(headers)(test.handler)

			req, err := http.NewRequest(http.MethodGet, "/test", nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)
			res := w.Result()
			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Fatalf("failed to close response body: %v", err)
				}
			}()

			assert.Equal(
				t,
				"value",
				res.Header.Get("x-key"),
				"they should be equal",
			)
		})
	}
}
