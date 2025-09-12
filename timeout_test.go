package rmhttp

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// TIMEOUT TESTS
// ------------------------------------------------------------------------------------------------

// Test_Timeout_applyTimeout checks that route timeouts act as expected
func Test_Timeout_applyTimeout(t *testing.T) {
	testPattern := "/timeout"
	timeoutMessage := "Timeout!!!"
	testBody := "Timeout body"
	url := fmt.Sprintf("http://%s%s", testAddress, testPattern)

	t.Run("timeout handler intercepts long running handler and throws error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			_, _ = w.Write([]byte("Hello"))
		})

		// Create a request that would trigger our test handler
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Errorf("failed to create request: %v", err)
		}

		w := httptest.NewRecorder()
		route := NewRoute(
			http.MethodGet,
			testPattern,
			handler,
		).WithTimeout(1*time.Second, timeoutMessage)
		middleware := []func(http.Handler) http.Handler{TimeoutMiddleware(route.ComputedTimeout())}
		h := applyMiddleware(
			route.Handler,
			middleware,
		)
		h.ServeHTTP(w, req)

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

		assert.Equal(t, http.StatusServiceUnavailable, res.StatusCode, "they should be equal")
		assert.Equal(t, timeoutMessage, string(body), "they should be equal")
	})

	t.Run("timeout handler passes through handler without error", func(t *testing.T) {
		handler := http.HandlerFunc(createTestHandlerFunc(http.StatusOK, testBody))
		route := NewRoute(
			http.MethodGet,
			testPattern,
			handler,
		).WithTimeout(1*time.Second, timeoutMessage)

		// Create a request that would trigger our test handler
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Errorf("failed to create request: %v", err)
		}

		w := httptest.NewRecorder()
		route.Handler.ServeHTTP(w, req)
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

		assert.Equal(t, testBody, string(body), "they should be equal")
		assert.Equal(t, http.StatusOK, res.StatusCode, "they should be equal")
	})
}
