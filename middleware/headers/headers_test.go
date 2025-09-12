package headers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// RECOVERER TESTS
// ------------------------------------------------------------------------------------------------

const (
	testAddress string = "localhost:8123"
)

// Test_Recoverer checks that a panic thrown within a request can be recovered from, and then
// return an appropriate error.
func Test_Headers(t *testing.T) {
	testPattern := "/test"

	tests := []struct {
		name          string
		expectedKey   string
		expectedValue string
		handler       http.Handler
	}{
		{
			"Headers are added to response",
			"x-key",
			"key",
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

			url := fmt.Sprintf("http://%s%s", testAddress, testPattern)
			req, err := http.NewRequest(http.MethodGet, url, nil)
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
