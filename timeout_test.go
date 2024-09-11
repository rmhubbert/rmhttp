package rmhttp

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_Timeout_applyTimeout checks that route timeouts act as expected
func Test_Timeout_applyTimeout(t *testing.T) {
	testPattern := "/timeout"
	timeoutMessage := "Timeout!!!"
	testBody := "Timeout body"
	url := fmt.Sprintf("http://%s%s", testAddress, testPattern)

	t.Run("timeout handler intercepts long running handler and returns error", func(t *testing.T) {
		handler := HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			time.Sleep(2 * time.Second)
			return nil
		})

		// Create a request that would trigger our test handler
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Errorf("failed to create request: %v", err)
		}

		w := httptest.NewRecorder()
		route := NewRoute(http.MethodGet, testPattern, handler)
		timeout := NewTimeout(1*time.Second, timeoutMessage)
		route.Handler = NewTimeoutHandler(handler, timeout)

		timeoutErr := route.Handler.ServeHTTPWithError(w, req)
		te := timeoutErr.(HTTPError)

		assert.Equal(t, http.StatusServiceUnavailable, te.Code, "they should be equal")
		assert.Equal(t, timeoutMessage, te.Err.Error(), "they should be equal")
	})

	t.Run("timeout handler passes through handler without error", func(t *testing.T) {
		handler := HandlerFunc(createTestHandlerFunc(http.StatusOK, testBody, nil))
		route := NewRoute(http.MethodGet, testPattern, handler)
		timeout := NewTimeout(1*time.Second, timeoutMessage)
		route.Handler = NewTimeoutHandler(handler, timeout)

		// Create a request that would trigger our test handler
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Errorf("failed to create request: %v", err)
		}

		w := httptest.NewRecorder()
		timeoutErr := route.Handler.ServeHTTPWithError(w, req)
		res := w.Result()
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("failed to read response body: %v", err)
		}

		require.NoError(t, timeoutErr, "there should be no error")
		assert.Equal(t, testBody, string(body), "they should be equal")
		assert.Equal(t, http.StatusOK, res.StatusCode, "they should be equal")
	})
}
