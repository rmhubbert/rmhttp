package apikey

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
func Test_ApiKey(t *testing.T) {
	testPattern := "/test"
	keys := []string{
		"apikey",
		"apikey2",
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name         string
		expectedCode int
		setApiKey    bool
		apiKey       string
		handler      http.Handler
	}{
		{
			"a 200 response is returned when a valid x-api-key header is sent",
			http.StatusOK,
			true,
			"apikey",
			handler,
		},
		{
			"a 200 response is returned when a different valid x-api-key header is sent",
			http.StatusOK,
			true,
			"apikey2",
			handler,
		},
		{
			"a 401 response is returned when an invalid x-api-key header is sent",
			http.StatusUnauthorized,
			true,
			"invalid",
			handler,
		},
		{
			"a 401 response is returned when an empty x-api-key header is sent",
			http.StatusUnauthorized,
			true,
			"",
			handler,
		},
		{
			"a 401 response is returned when no x-api-key header is sent",
			http.StatusUnauthorized,
			false,
			"",
			handler,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := Middleware(keys...)(test.handler)

			url := fmt.Sprintf("http://%s%s", testAddress, testPattern)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			if test.setApiKey {
				req.Header.Set("x-api-key", test.apiKey)
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

			assert.Equal(t, test.expectedCode, res.StatusCode, "they should be equal")
		})
	}
}
