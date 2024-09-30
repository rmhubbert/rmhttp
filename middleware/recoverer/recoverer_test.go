package recoverer

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rmhubbert/rmhttp"
	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// RECOVERER TESTS
// ------------------------------------------------------------------------------------------------

const (
	testAddress string = "localhost:8080"
)

// Test_Recoverer checks that a panic thrown within a request can be recovered from, and then
// return an appropriate error.
func Test_Recoverer(t *testing.T) {
	testPattern := "/test"

	tests := []struct {
		name          string
		expectedCode  int
		panicExpected bool
		route         *rmhttp.Route
	}{
		{
			"a panic is recovered from and the response status is set to 500",
			http.StatusInternalServerError,
			true,
			rmhttp.NewRoute(
				http.MethodGet,
				testPattern,
				rmhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
					panic("Paniced!")
				}),
			),
		},
		{
			"Recoverer does nothing when no panic has occurred",
			http.StatusOK,
			false,
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
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			_ = test.route.Handler.ServeHTTPWithError(w, req)
			res := w.Result()
			defer res.Body.Close()

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
