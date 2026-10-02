package rmhttp

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// ROUTER ALLOCATION BUDGET TESTS
//
// Budgets are allocation-free-handler measurements at the router level, asserted relative to a
// bare http.ServeMux serving the identical pattern and handler. The reused-request group below
// serves one request repeatedly (the routing cost on a keep-alive connection after path-value
// state exists); the fresh-request group measures production-shaped requests, where hidden costs
// such as SetPathValue's backing-map allocation surface.
// ------------------------------------------------------------------------------------------------

// noopWriter is an allocation-free sink ResponseWriter for the budget measurements. The persistent
// header map keeps Header() allocation-free should a handler touch it.
type noopWriter struct {
	header http.Header
	status int
}

func newNoopWriter() *noopWriter {
	return &noopWriter{header: make(http.Header, 4), status: http.StatusOK}
}

func (w *noopWriter) Header() http.Header { return w.header }

func (w *noopWriter) Write(p []byte) (int, error) { return len(p), nil }

func (w *noopWriter) WriteHeader(status int) { w.status = status }

// captureValueHandler reads the named path values and writes nothing, so the measured allocations
// belong to the router rather than the test fixture.
func captureValueHandler(names ...string) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		for _, name := range names {
			_ = r.PathValue(name)
		}
	})
}

// intercept enables the Router's 404/405 interception path. Without a registered error handler
// the Router hands every request straight to the mux, so the interception fast path under test
// in the budgets below must be activated the same way an App does with a custom error handler.
// The handler itself never runs for matched routes, so its body does not matter here.
func intercept(router *Router) *Router {
	router.AddErrorHandler(http.StatusNotFound, http.NotFoundHandler())
	return router
}

// measureServeAllocs serves req through h on a reused writer and returns allocations per run.
// Reusing the request hides first-call costs (see the fresh-request budgets below), which is
// exactly why this harness measures against a stdlib baseline run through the same reused shape.
func measureServeAllocs(t *testing.T, h http.Handler, req *http.Request) float64 {
	t.Helper()

	w := newNoopWriter()
	h.ServeHTTP(w, req) // warm lazily initialized state (path values, pools)
	runtime.GC()

	return testing.AllocsPerRun(200, func() {
		w.status = http.StatusOK
		clear(w.header)
		h.ServeHTTP(w, req)
	})
}

// assertReusedBudget serves one reused request shape through an intercepted Router and a bare
// ServeMux carrying the identical pattern and handler, and asserts the Router's allocations stay
// within allowance over the mux. The allowance covers the interception probe plus whatever the
// chosen dispatch style costs; the reasoning belongs in each call site's comment.
func assertReusedBudget(
	t *testing.T,
	method, pattern string,
	allowance float64,
	reason string,
	handler http.Handler,
) {
	t.Helper()

	stdMux := http.NewServeMux()
	stdMux.Handle(method+" "+pattern, handler)

	router := intercept(NewRouter())
	router.Handle(method, pattern, handler)

	req := httptest.NewRequest(method, patternPath(pattern), nil)

	stdlib := measureServeAllocs(t, stdMux, req)
	rmhttp := measureServeAllocs(t, router, req)

	t.Logf("%s %s: stdlib=%.0f rmhttp=%.0f (allowance %.0f: %s)",
		method, pattern, stdlib, rmhttp, allowance, reason)
	assert.LessOrEqual(
		t,
		rmhttp,
		stdlib+allowance,
		"%s %s must stay within its allocation allowance over stdlib",
		method,
		pattern,
	)
}

// patternPath turns a pattern like "/users/{id}/posts/{post_id}" into a path that matches it.
func patternPath(pattern string) string {
	var b strings.Builder
	for {
		open := strings.IndexByte(pattern, '{')
		if open < 0 {
			b.WriteString(pattern)
			return b.String()
		}
		end := strings.IndexByte(pattern[open:], '}')
		b.WriteString(pattern[:open])
		b.WriteString("123")
		pattern = pattern[open+end+1:]
	}
}

// Test_AllocBudget_MatchedRequest budgets a two-param matched request. One and two wildcard
// patterns re-match through the mux under interception so their path values are filled natively;
// that choice trades two reused-request match costs for fresh-request parity, so the allowance
// here covers the discarded probe match on top of the mux's own pass.
func Test_AllocBudget_MatchedRequest(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	assertReusedBudget(
		t,
		http.MethodGet,
		"/users/{id}/posts/{post_id}",
		2,
		"probe re-match buys fresh-request native values",
		captureValueHandler("id", "post_id"),
	)
}

// Test_AllocBudget_PathValue budgets a three-param matched request. Three or more wildcards stay
// on direct dispatch: the mux's second match costs more there than re-applying the values the
// probe found, so only the SetPathValue backing map is extra over stdlib — and a reused request
// already has that map, leaving full parity.
func Test_AllocBudget_PathValue(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	assertReusedBudget(
		t,
		http.MethodGet,
		"/a/{x}/b/{y}/c/{z}",
		0,
		"direct dispatch is reused-request parity",
		captureValueHandler("x", "y", "z"),
	)
}

// Test_AllocBudget_WildcardTail budgets a wildcard-tail matched request. Tail patterns stay on
// direct dispatch because the mux's own tail match is its most expensive shape; a reused request
// pays nothing extra for the re-applied value.
func Test_AllocBudget_WildcardTail(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	assertReusedBudget(
		t,
		http.MethodGet,
		"/files/{path...}",
		0,
		"direct dispatch is reused-request parity",
		captureValueHandler("path"),
	)
}

// Test_AllocBudget_ExactRoute pins that plain exact routes stay allocation-free under
// interception (the probe costs nothing on exact matches and dispatch is direct).
func Test_AllocBudget_ExactRoute(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	assertReusedBudget(
		t,
		http.MethodGet,
		"/exact",
		0,
		"exact routes must remain allocation-free",
		captureValueHandler(),
	)
}

// ------------------------------------------------------------------------------------------------
// FRESH-REQUEST ALLOCATION BUDGETS
//
// The budgets above reuse one request, which hides allocations that only happen when a request
// carries no path-value state yet: the first SetPathValue on a fresh request allocates a backing
// map, and a serving path that re-derives values pays that on every request. These budgets serve
// a newly constructed request each iteration and subtract the request-construction cost, pinning
// the production-shaped delta over a bare http.ServeMux doing the same work.
// ------------------------------------------------------------------------------------------------

// freshRequestSink keeps the constructed request alive so the construction baseline measures the
// same heap allocations the serving measurement pays for.
var freshRequestSink *http.Request

// newFreshRequest builds a minimal production-shaped GET request, deriving Path and RawPath the
// way http.ReadRequest does so EscapedPath round-trips escaped input.
func newFreshRequest(escapedPath string) *http.Request {
	u, err := url.ParseRequestURI(escapedPath)
	if err != nil {
		panic(err)
	}
	return &http.Request{
		Method:     http.MethodGet,
		URL:        u,
		RequestURI: escapedPath,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Host:       "example.com",
	}
}

// freshHandlerAllocs returns the allocations per request that h costs on top of building a fresh
// request for escapedPath, measured the same way for any handler under test.
func freshHandlerAllocs(t *testing.T, h http.Handler, escapedPath string) float64 {
	t.Helper()

	w := newNoopWriter()
	h.ServeHTTP(w, newFreshRequest(escapedPath)) // warm lazily initialized state (pools)
	runtime.GC()

	base := testing.AllocsPerRun(200, func() {
		freshRequestSink = newFreshRequest(escapedPath)
	})

	return testing.AllocsPerRun(200, func() {
		req := newFreshRequest(escapedPath)
		w.status = http.StatusOK
		h.ServeHTTP(w, req)
	}) - base
}

// freshShape describes one route shape and the allocation headroom the router is allowed over a
// bare ServeMux serving the same pattern and handler.
type freshShape struct {
	pattern   string
	path      string
	params    []string
	allowance float64
	why       string
}

// runFreshBudgets measures each shape against the same pattern on a bare http.ServeMux and
// asserts the router's extra allocations stay within the per-shape allowance.
func runFreshBudgets(t *testing.T, shapes []freshShape, intercept bool) {
	t.Helper()

	for _, shape := range shapes {
		handler := captureValueHandler(shape.params...)

		stdMux := http.NewServeMux()
		method, pattern, _ := strings.Cut(shape.pattern, " ")
		stdMux.Handle(shape.pattern, handler)

		router := NewRouter()
		router.Handle(method, pattern, handler)
		if intercept {
			// Registering an error handler re-enables the interception probe, which is the shape
			// every App with custom or default 404/405 handlers serves in.
			router.AddErrorHandler(http.StatusNotFound, http.NotFoundHandler())
		}

		stdlib := freshHandlerAllocs(t, stdMux, shape.path)
		rmhttp := freshHandlerAllocs(t, router, shape.path)

		t.Logf("%-26s stdlib=%.0f rmhttp=%.0f (allowance %.0f: %s)",
			shape.pattern, stdlib, rmhttp, shape.allowance, shape.why)
		assert.LessOrEqual(
			t,
			rmhttp,
			stdlib+shape.allowance,
			"%s must stay within its fresh-request allocation allowance over stdlib",
			shape.pattern,
		)
	}
}

// Test_AllocBudget_FreshRequest_MuxPassthrough pins pure-stdlib parity when nothing is registered
// to intercept: a bare Router — and an App that registers no error handlers, the default — must
// cost exactly what a bare ServeMux costs on fresh requests.
func Test_AllocBudget_FreshRequest_MuxPassthrough(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	runFreshBudgets(t, []freshShape{
		{"GET /exact", "/exact", nil, 0, "mux passthrough is the mux"},
		{"GET /users/{id}", "/users/123", []string{"id"}, 0, "mux populates path values natively"},
		{
			"GET /a/{x}/b/{y}",
			"/a/1/b/2",
			[]string{"x", "y"},
			0,
			"mux populates path values natively",
		},
		{
			"GET /a/{x}/b/{y}/c/{z}",
			"/a/1/b/2/c/3",
			[]string{"x", "y", "z"},
			0,
			"mux populates path values natively",
		},
		{
			"GET /files/{path...}",
			"/files/path/to/file.txt",
			[]string{"path"},
			0,
			"mux populates path values natively",
		},
	}, false)
}

// Test_AllocBudget_FreshRequest_Interception pins the headroom of the interception path: the mux
// Handler probe costs roughly one discarded match per wildcard, and patterns kept on direct
// dispatch (three or more wildcards, trailing "{name...}") pay the SetPathValue backing map. One
// or two wildcard patterns re-match through the mux instead, so only the probe is extra.
func Test_AllocBudget_FreshRequest_Interception(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	runFreshBudgets(t, []freshShape{
		{"GET /exact", "/exact", nil, 1, "probe is free on exact matches"},
		{"GET /users/{id}", "/users/123", []string{"id"}, 1, "one discarded probe match"},
		{"GET /a/{x}/b/{y}", "/a/1/b/2", []string{"x", "y"}, 2, "two discarded probe matches"},
		{
			"GET /a/{x}/b/{y}/c/{z}",
			"/a/1/b/2/c/3",
			[]string{"x", "y", "z"},
			2,
			"probe plus SetPathValue map beats a third mux match",
		},
		{
			"GET /files/{path...}",
			"/files/path/to/file.txt",
			[]string{"path"},
			2,
			"probe plus SetPathValue map beats a mux tail re-match",
		},
	}, true)
}
