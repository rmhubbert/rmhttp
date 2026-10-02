package rmhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// ERROR-PATH FUNCTIONAL PARITY GATES (Phase 3)
//
// These tests pin the observable 404/405 responses (status, headers, exact bytes of the body) and
// the "error handler executes exactly once" contract. They pass against the pre-Phase-3
// implementation and must stay passing through the single-execution rewrite.
// ------------------------------------------------------------------------------------------------

// errorResponder serves a request through a compiled App and exposes the recorded response.
func errorResponder(t *testing.T, app *App, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	app.Compile()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	return w
}

func Test_Router_CustomErrorHandlerRunsExactlyOnce(t *testing.T) {
	var notFoundCalls, methodCalls int

	app := New()
	app.Get("/exists", func(w http.ResponseWriter, _ *http.Request) {})
	app.StatusNotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
		notFoundCalls++
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("custom 404"))
	})
	app.StatusMethodNotAllowedHandler(func(w http.ResponseWriter, _ *http.Request) {
		methodCalls++
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("custom 405"))
	})

	w := errorResponder(t, app, http.MethodGet, "/missing")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, "custom 404", w.Body.String())
	assert.Equal(t, 1, notFoundCalls, "the custom 404 handler must run exactly once")
	assert.Zero(t, methodCalls, "the 405 handler must not fire on a 404")
}

func Test_Router_CustomMethodHandlerRunsExactlyOnce(t *testing.T) {
	var notFoundCalls, methodCalls int

	app := New()
	app.Get("/exists", func(w http.ResponseWriter, _ *http.Request) {})
	app.StatusNotFoundHandler(func(w http.ResponseWriter, _ *http.Request) {
		notFoundCalls++
	})
	app.StatusMethodNotAllowedHandler(func(w http.ResponseWriter, _ *http.Request) {
		methodCalls++
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("custom 405"))
	})

	w := errorResponder(t, app, http.MethodPost, "/exists")

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	assert.Equal(t, "custom 405", w.Body.String())
	assert.Equal(t, "GET, HEAD", w.Header().Get("Allow"))
	assert.Zero(t, notFoundCalls, "the 404 handler must not fire on a 405")
	assert.Equal(t, 1, methodCalls, "the custom 405 handler must run exactly once")
}

// Test_Router_BareRouterErrorReplay pins the no-custom-handler case: a bare Router without
// registered error handlers must replay the stdlib error response byte-for-byte, exactly once.
func Test_Router_BareRouterErrorReplay(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantAllow  string
		wantBody   string
	}{
		{
			name:       "stdlib 404 replay",
			method:     http.MethodGet,
			path:       "/missing",
			wantStatus: http.StatusNotFound,
			wantBody:   "404 page not found\n",
		},
		{
			name:       "stdlib 405 replay",
			method:     http.MethodPost,
			path:       "/exists",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  "GET, HEAD",
			wantBody:   "Method Not Allowed\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := NewRouter()
			router.Handle(http.MethodGet, "/exists", http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {},
			))

			req := httptest.NewRequest(test.method, test.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, test.wantStatus, w.Code)
			assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
			assert.Equal(t, test.wantAllow, w.Header().Get("Allow"))
			assert.Equal(t, test.wantBody, w.Body.String())
		})
	}
}
