package rmhttp

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// ERROR-PATH ALLOCATION BUDGET TESTS (Phase 3 red/green gates)
//
// Budgets equal the Phase-1 post-harness floors (benchmarks/step1-harness/alloc-floor.txt) minus
// the cost of the double handler execution the current implementation pays: the mux's error
// handler is executed once into the CaptureWriter and, when no custom handler exists, a second
// time against the real writer; even with a custom handler, the default seeded handlers allocate
// their status-text body per response. They fail now and must pass from T19-T21 onward.
// ------------------------------------------------------------------------------------------------

// measureErrorAllocs serves an error request through a compiled App on a reused writer.
func measureErrorAllocs(t *testing.T, app *App, req *http.Request) float64 {
	t.Helper()

	w := newNoopWriter()
	app.Router.ServeHTTP(w, req) // warm pools and lazily initialized state
	runtime.GC()

	return testing.AllocsPerRun(200, func() {
		w.status = http.StatusOK
		clear(w.header)
		app.Router.ServeHTTP(w, req)
	})
}

// Test_AllocBudget_404 budgets the default 404 path at 12: Phase-1 floor 13 minus the one
// allocation the double handler execution owns on the App flow (the []byte(StatusText)
// conversion), which the precomputed default body removes.
func Test_AllocBudget_404(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure",
		)
	}

	app := New()
	app.Get("/exists", func(_ http.ResponseWriter, _ *http.Request) {})
	app.Compile()

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	allocs := measureErrorAllocs(t, app, req)

	t.Logf("measured allocations per 404: %.0f (budget 12)", allocs)
	assert.LessOrEqual(t, allocs, 12.0, "404 must not exceed the single-execution budget")
}

// Test_AllocBudget_405 budgets the default 405 path at 14: Phase-1 floor 15 minus the
// double-execution allocation removed by the precomputed default body.
func Test_AllocBudget_405(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure",
		)
	}

	app := New()
	app.Get("/exists", func(_ http.ResponseWriter, _ *http.Request) {})
	app.Compile()

	req := httptest.NewRequest(http.MethodPost, "/exists", nil)
	allocs := measureErrorAllocs(t, app, req)

	t.Logf("measured allocations per 405: %.0f (budget 14)", allocs)
	assert.LessOrEqual(t, allocs, 14.0, "405 must not exceed the single-execution budget")
}
