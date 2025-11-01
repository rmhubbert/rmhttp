package recoverer

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
func Test_Recoverer(t *testing.T) {
	testPattern := "/test"

	tests := []struct {
		name          string
		expectedCode  int
		panicExpected bool
		handler       http.Handler
	}{
		{
			"a panic is recovered from and the response status is set to 500",
			http.StatusInternalServerError,
			true,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic("Panicked!")
			}),
		},
		{
			"Recoverer does nothing when no panic has occurred",
			http.StatusOK,
			false,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := Middleware()(test.handler)

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

			if test.panicExpected {
				assert.Equal(
					t,
					http.StatusInternalServerError,
					res.StatusCode,
					"they should be equal",
				)
			} else {
				assert.Equal(t, http.StatusOK, res.StatusCode, "they should be equal")
			}
		})
	}
}
