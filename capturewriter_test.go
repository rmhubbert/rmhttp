package rmhttp

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CaptureWriter_Persist(t *testing.T) {
	testAddress := "localhost:8123"
	testPattern := "/persist"
	testBody := "persist"
	handler := http.HandlerFunc(createTestHandlerFunc(http.StatusOK, testBody))

	h := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cw := NewCaptureWriter(w)

			cw.Header().Add("x-pre", "pre")
			next.ServeHTTP(cw, r)
			// cw.Header().Add("x-post", "post")
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
	// assert.Equal(t, "post", res.Header.Get("x-post"), "they should be equal")
}

func Test_CaptureWriter_DefaultStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	cw := NewCaptureWriter(w)

	assert.Equal(t, http.StatusOK, cw.Code, "Code should default to 200")
}

func Test_CaptureWriter_CapturesStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	cw := NewCaptureWriter(w)

	cw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, cw.Code, "captured status should match")
	assert.Equal(t, http.StatusNotFound, w.Code, "underlying writer status should match")
}

func Test_CaptureWriter_AccumulatesBodyChunks(t *testing.T) {
	w := httptest.NewRecorder()
	cw := NewCaptureWriter(w)

	// Simulate multiple Write calls (e.g., from json.Encoder, io.Copy, etc.)
	cw.Write([]byte("hello"))
	cw.Write([]byte(" "))
	cw.Write([]byte("world"))

	assert.Equal(t, "hello world", cw.Body, "body should accumulate all chunks")
	assert.Equal(t, "hello world", w.Body.String(), "underlying writer should have full body")
}

func Test_CaptureWriter_PassThroughFalseSkipsBodyCapture(t *testing.T) {
	w := httptest.NewRecorder()
	cw := NewCaptureWriter(w)
	cw.PassThrough = false

	cw.WriteHeader(http.StatusNoContent)
	cw.Write([]byte("should not be captured"))

	assert.Equal(t, http.StatusNoContent, cw.Code, "status should still be captured")
	assert.Empty(t, cw.Body, "body should not be captured when PassThrough is false")
	assert.Empty(t, w.Body.String(), "underlying writer should not receive body")
}

func Test_CaptureWriter_PassThroughFalseSkipsWriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	cw := NewCaptureWriter(w)
	cw.PassThrough = false

	cw.WriteHeader(http.StatusInternalServerError)

	assert.Equal(t, http.StatusInternalServerError, cw.Code, "status should still be captured")
	assert.Equal(t, http.StatusOK, w.Code, "underlying writer should not receive WriteHeader")
}

func Test_CaptureWriter_CapturesStatusCodeAndBody(t *testing.T) {
	w := httptest.NewRecorder()
	cw := NewCaptureWriter(w)

	cw.WriteHeader(http.StatusBadRequest)
	cw.Write([]byte("bad request"))

	assert.Equal(t, http.StatusBadRequest, cw.Code, "captured status should match")
	assert.Equal(t, "bad request", cw.Body, "captured body should match")
}
