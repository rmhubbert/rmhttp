package rmhttp

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CaptureWriter_Persist(t *testing.T) {
	testPattern := "/persist"
	testBody := "persist"
	route := NewRoute(
		http.MethodGet,
		testPattern,
		HandlerFunc(createTestHandlerFunc(http.StatusOK, testBody, nil)),
	)

	count := 3
	for i := 0; i < count; i++ {
		route.Use(func(next Handler) Handler {
			return HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
				cw := NewCaptureWriter(w)
				defer cw.Persist()

				cw.Header().Add("x-pre-"+strconv.Itoa(i), "pre-"+strconv.Itoa(i))
				err := next.ServeHTTPWithError(cw, r)
				cw.Header().Add("x-post-"+strconv.Itoa(i), "post-"+strconv.Itoa(i))

				return err
			})
		})
	}

	handler := applyMiddleware(route.Handler, route.ComputedMiddleware())
	mux := http.NewServeMux()
	mux.Handle(fmt.Sprintf("%s %s", route.Method, route.ComputedPattern()), handler)

	// Create a request that would trigger our test handler
	url := fmt.Sprintf("http://%s%s", testAddress, testPattern)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Errorf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	_ = handler.ServeHTTPWithError(w, req)
	res := w.Result()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("failed to read response body: %v", err)
	}

	assert.Equal(t, http.StatusOK, res.StatusCode, "they should be equal")
	assert.Equal(t, testBody, string(body), "they should be equal")

	for i := 0; i < count; i++ {
		assert.Equal(
			t,
			"pre-"+strconv.Itoa(i),
			res.Header.Get("x-pre-"+strconv.Itoa(i)),
			"they should be equal",
		)
		assert.Equal(
			t,
			"post-"+strconv.Itoa(i),
			res.Header.Get("x-post-"+strconv.Itoa(i)),
			"they should be equal",
		)
	}
}
