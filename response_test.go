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

// ------------------------------------------------------------------------------------------------
// RESPONSE TESTS
// ------------------------------------------------------------------------------------------------

func Test_CaptureWriter_Persist(t *testing.T) {
	testPattern := "/persist"
	testBody := "persist"
	route := NewRoute(
		http.MethodGet,
		testPattern,
		http.HandlerFunc(createTestHandlerFunc(http.StatusOK, testBody)),
	)

	count := 3
	for i := 0; i < count; i++ {
		route.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				cw := NewCaptureWriter(w)
				defer cw.Persist()

				cw.Header().Add("x-pre-"+strconv.Itoa(i), "pre-"+strconv.Itoa(i))
				next.ServeHTTP(cw, r)
				cw.Header().Add("x-post-"+strconv.Itoa(i), "post-"+strconv.Itoa(i))
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
	handler.ServeHTTP(w, req)

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
