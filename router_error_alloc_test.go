package rmhttp

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// ERROR-PATH ALLOCATION BUDGET TESTS
//
// Budgets equal the Phase-1 pre-harness floors (benchmarks/step1-harness/alloc-floor.txt): the
// default App registers no error handlers, so 404 and 405 are served by the mux exactly as plain
// net/http serves them, and the budgets pin that parity ceiling.
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

// Test_AllocBudget_404 budgets the default 404 path at 12: the Phase-1 floor of 13, which the
// mux-native path now matches because the App registers no handler to intercept.
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

// Test_AllocBudget_405 budgets the default 405 path at 14: the Phase-1 floor of 15, for the
// same reason as the 404 budget above.
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
