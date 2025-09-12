package capturewriter

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestHandlerFunc(
	status int,
	body string,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func Test_CaptureWriter_Persist(t *testing.T) {
	testAddress := "localhost:8123"
	testPattern := "/persist"
	testBody := "persist"
	handler := http.HandlerFunc(createTestHandlerFunc(http.StatusOK, testBody))

	h := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cw := New(w)
			defer cw.Persist()

			cw.Header().Add("x-pre", "pre")
			next.ServeHTTP(cw, r)
			cw.Header().Add("x-post", "post")
		})
	}(handler)

	// Create a request that would trigger our test handler
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
			t.Errorf("failed to close request body: %v", err)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("failed to read response body: %v", err)
	}

	assert.Equal(t, http.StatusOK, res.StatusCode, "they should be equal")
	assert.Equal(t, testBody, string(body), "they should be equal")
	assert.Equal(t, "pre", res.Header.Get("x-pre"), "they should be equal")
	assert.Equal(t, "post", res.Header.Get("x-post"), "they should be equal")
}
