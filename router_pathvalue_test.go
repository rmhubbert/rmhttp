package rmhttp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// PATHVALUE FUNCTIONAL GATES
//
// These tests pin the request-visible behavior of the router hot path: r.PathValue must return
// exactly what the stdlib ServeMux provides for the matched pattern. They pass against the
// pre-Phase-2 implementation and must stay passing through every later phase.
// ------------------------------------------------------------------------------------------------

func Test_Router_PathValue_SingleSegment(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected string
	}{
		{"plain segment", "/users/{id}", "/users/123", "123"},
		{"multi-character segment", "/users/{id}", "/users/alice", "alice"},
		{"escaped space decodes to space", "/users/{id}", "/users/a%20b", "a b"},
		{"escaped slash stays within one segment", "/users/{id}", "/users/a%2Fb", "a/b"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := NewRouter()
			var got string
			router.Handle(http.MethodGet, test.pattern, http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					got = r.PathValue("id")
				},
			))

			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, test.expected, got, "PathValue must match stdlib unescaping semantics")
		})
	}
}

func Test_Router_PathValue_Wildcard(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		path     string
		expected string
	}{
		{"multi-segment tail", "/files/{path...}", "/files/a/b/c.txt", "a/b/c.txt"},
		{"single-segment tail", "/files/{path...}", "/files/one.txt", "one.txt"},
		{"tail with escaped space", "/files/{path...}", "/files/a%20b/c", "a b/c"},
		{"tail keeps real slashes", "/files/{path...}", "/files/x/y/z", "x/y/z"},
		{"tail preserves trailing slash", "/files/{path...}", "/files/a/b/", "a/b/"},
		{"empty tail is empty string", "/files/{path...}", "/files/", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := NewRouter()
			var got string
			router.Handle(http.MethodGet, test.pattern, http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					got = r.PathValue("path")
				},
			))

			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(
				t,
				test.expected,
				got,
				"wildcard PathValue must capture the unescaped tail",
			)
		})
	}
}

func Test_Router_PathValue_MultipleParams(t *testing.T) {
	router := NewRouter()
	var version, id string
	router.Handle(http.MethodGet, "/api/{version}/users/{id}", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			version = r.PathValue("version")
			id = r.PathValue("id")
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/v2/users/42", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "v2", version)
	assert.Equal(t, "42", id)
}

// Test_Router_PathValue_HEAD covers the HEAD-falls-back-to-GET behavior of the mux: the handler
// registered for GET must still see its path values when served a HEAD request.
func Test_Router_PathValue_HEAD(t *testing.T) {
	router := NewRouter()
	var got string
	router.Handle(http.MethodGet, "/users/{id}", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			got = r.PathValue("id")
		},
	))

	req := httptest.NewRequest(http.MethodHead, "/users/99", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "99", got)
}

// Test_Router_PathValue_MethodSpecific pins that a method-specific registration serves only that
// method and the wrong method takes the error path, still with correct extraction on the match.
func Test_Router_PathValue_MethodSpecific(t *testing.T) {
	router := NewRouter()
	var got string
	router.Handle(http.MethodPost, "/things/{id}", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			got = r.PathValue("id")
		},
	))

	postReq := httptest.NewRequest(http.MethodPost, "/things/7", nil)
	postW := httptest.NewRecorder()
	router.ServeHTTP(postW, postReq)
	assert.Equal(t, http.StatusOK, postW.Code)
	assert.Equal(t, "7", got)

	getReq := httptest.NewRequest(http.MethodGet, "/things/7", nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)
	assert.Equal(t, http.StatusMethodNotAllowed, getW.Code)
	assert.Equal(t, "7", got, "the 405 dispatch must not clobber the earlier successful extraction")
}

// Test_Router_RedirectPreserved pins stdlib canonical-path redirects: a request for /tree where
// only /tree/ is registered must 301/307-redirect exactly like the bare ServeMux, including the
// Location header.
func Test_Router_RedirectPreserved(t *testing.T) {
	router := NewRouter()
	router.Handle(http.MethodGet, "/tree/", http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("tree index"))
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/tree", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/tree/", w.Header().Get("Location"))
}

// Test_Router_UncleanPathRedirect pins the cleaned-path redirect: an unclean request path that
// would match after cleaning must redirect to the canonical path rather than hit the handler.
func Test_Router_UncleanPathRedirect(t *testing.T) {
	router := NewRouter()
	var got string
	router.Handle(http.MethodGet, "/users/{id}", http.HandlerFunc(
		func(_ http.ResponseWriter, r *http.Request) {
			got = r.PathValue("id")
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/users//123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
	assert.Equal(t, "/users/123", w.Header().Get("Location"))
	assert.Empty(t, got, "the handler must not run for the unclean path")
}

// Test_Router_OptionStarPreserved pins the RequestURI == "*" special case: the mux answers 400
// with Connection: close and must keep doing so through the Router.
func Test_Router_OptionStarPreserved(t *testing.T) {
	router := NewRouter()
	called := false
	router.Handle(http.MethodOptions, "/", http.HandlerFunc(
		func(_ http.ResponseWriter, _ *http.Request) {
			called = true
		},
	))

	req := httptest.NewRequest(http.MethodOptions, "*", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "close", w.Header().Get("Connection"))
	assert.False(t, called, "the OPTIONS * request must not reach any registered handler")
}

// Test_Router_PathValue_Intercepted pins that both interception dispatch styles hand handlers the
// same path values a bare ServeMux would: one and two wildcard patterns re-match through the mux
// (nativeMatch), while three-plus wildcards and trailing "{name...}" re-apply the probe's match
// with SetPathValue. Escaped inputs pin the unescaping behavior of both.
func Test_Router_PathValue_Intercepted(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		param   string
		want    string
	}{
		{"/users/{id}", "/users/123", "id", "123"},
		{"/users/{id}", "/users/a%20b", "id", "a b"},
		{"/users/{id}", "/users/a%2Fb", "id", "a/b"},
		{"/api/{v}/users/{id}", "/api/v2/users/42", "v", "v2"},
		{"/api/{v}/users/{id}", "/api/v2/users/42", "id", "42"},
		{"/a/{x}/b/{y}/c/{z}", "/a/1/b/2/c/3", "z", "3"},
		{"/a/{x}/b/{y}/c/{z}", "/a/%41/b/2/c/3", "x", "A"},
		{"/files/{path...}", "/files/a/b/c.txt", "path", "a/b/c.txt"},
		{"/files/{path...}", "/files/a%20b/c", "path", "a b/c"},
	}

	for _, test := range tests {
		t.Run(test.pattern+test.path, func(t *testing.T) {
			router := intercept(NewRouter())
			var got string
			router.Handle(http.MethodGet, test.pattern, http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					got = r.PathValue(test.param)
				},
			))

			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(
				t,
				test.want,
				got,
				"PathValue must match stdlib semantics under interception",
			)
		})
	}
}

// Test_Router_PathValue_UnparseablePatterns pins that wildcard shapes the direct-dispatch fast
// path cannot re-derive — here, a wildcard before a trailing slash — are served by the mux
// natively so no path value is lost, under interception as well as without.
func Test_Router_PathValue_UnparseablePatterns(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		path    string
		param   string
		want    string
	}{
		{"wildcard before trailing slash", "/x/{id}/", "/x/7/", "id", "7"},
	}

	for _, interceptOn := range []bool{false, true} {
		for _, test := range tests {
			t.Run(fmt.Sprintf("%v/%s", interceptOn, test.name), func(t *testing.T) {
				router := NewRouter()
				if interceptOn {
					intercept(router)
				}
				var got string
				router.Handle(http.MethodGet, test.pattern, http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						got = r.PathValue(test.param)
					},
				))

				req := httptest.NewRequest(http.MethodGet, test.path, nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(
					t,
					test.want,
					got,
					"the mux must own path values the fast path cannot re-derive",
				)
			})
		}
	}
}
