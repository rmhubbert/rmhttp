package rmhttp

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// ROUTER ALLOCATION BUDGET TESTS (Phase 2 red/green gates)
//
// Budgets derive from the Phase-1 post-harness floors in benchmarks/step1-harness/alloc-floor.txt,
// probed at the router level with allocation-free handlers so the measurement covers routing only.
// Each budget equals (observed baseline − 1): the router must remove at least one full ServeMux
// match pass per request. They fail against the double-match implementation and must pass from
// T12 onward.
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

// measureServeAllocs serves req through router on a reused writer and returns allocations per run.
func measureServeAllocs(t *testing.T, router *Router, req *http.Request) float64 {
	t.Helper()

	w := newNoopWriter()
	router.ServeHTTP(w, req) // warm lazily initialized state (path values, pools)
	runtime.GC()

	return testing.AllocsPerRun(200, func() {
		w.status = http.StatusOK
		clear(w.header)
		router.ServeHTTP(w, req)
	})
}

// Test_AllocBudget_MatchedRequest budgets a two-param matched request at F-1 = 3 (baseline 4).
func Test_AllocBudget_MatchedRequest(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	router := NewRouter()
	router.Handle(
		http.MethodGet,
		"/users/{id}/posts/{post_id}",
		captureValueHandler("id", "post_id"),
	)

	req := httptest.NewRequest(http.MethodGet, "/users/123/posts/456", nil)
	allocs := measureServeAllocs(t, router, req)

	t.Logf("measured allocations per matched request: %.0f (budget 3)", allocs)
	assert.LessOrEqual(t, allocs, 3.0, "matched requests must not exceed the F-1 allocation budget")
}

// Test_AllocBudget_PathValue budgets a three-param matched request at F-1 = 5 (baseline 6).
func Test_AllocBudget_PathValue(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	router := NewRouter()
	router.Handle(http.MethodGet, "/a/{x}/b/{y}/c/{z}", captureValueHandler("x", "y", "z"))

	req := httptest.NewRequest(http.MethodGet, "/a/1/b/2/c/3", nil)
	allocs := measureServeAllocs(t, router, req)

	t.Logf("measured allocations per path-value request: %.0f (budget 5)", allocs)
	assert.LessOrEqual(
		t,
		allocs,
		5.0,
		"path-value requests must not exceed the F-1 allocation budget",
	)
}

// Test_AllocBudget_WildcardTail budgets a wildcard-tail matched request at F-2 = 8 (baseline 10).
// The tail pattern pays a stdlib trailing-slash probe per match pass, making it the most expensive
// matched shape; it benefits the most from removing one pass.
func Test_AllocBudget_WildcardTail(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	router := NewRouter()
	router.Handle(http.MethodGet, "/files/{path...}", captureValueHandler("path"))

	req := httptest.NewRequest(http.MethodGet, "/files/path/to/file.txt", nil)
	allocs := measureServeAllocs(t, router, req)

	t.Logf("measured allocations per wildcard-tail request: %.0f (budget 8)", allocs)
	assert.LessOrEqual(
		t,
		allocs,
		8.0,
		"wildcard-tail requests must not exceed the F-2 allocation budget",
	)
}

// Test_AllocBudget_ExactRoute pins that plain exact routes stay allocation-free after the change
// (they already are at baseline; this test guards the rewrite against regressions).
func Test_AllocBudget_ExactRoute(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	router := NewRouter()
	router.Handle(http.MethodGet, "/exact", captureValueHandler())

	req := httptest.NewRequest(http.MethodGet, "/exact", nil)
	allocs := measureServeAllocs(t, router, req)

	t.Logf("measured allocations per exact-route request: %.0f (budget 0)", allocs)
	assert.LessOrEqual(t, allocs, 0.0, "exact routes must remain allocation-free")
}
