package benchmarks

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/rmhubbert/rmhttp/v5"
	"github.com/stretchr/testify/assert"
)

// reuseTestBody is pre-allocated so the handler under test contributes no allocations; the
// measurement below is the writer acquire → serve → reset cycle itself.
var reuseTestBody = []byte("ok")

// Test_ReuseWriter_ZeroAllocs is the Phase-1 red/green gate for the benchmark harness: the
// reusable writer cycle that every hot-path benchmark relies on must not allocate at all, so
// measured allocations belong to rmhttp and not to the harness.
//
// The app uses an exact-match route (no wildcards): stdlib wildcard capture allocations are a
// separate concern, measured and attacked by the Phase-2 alloc-budget tests.
func Test_ReuseWriter_ZeroAllocs(t *testing.T) {
	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	app := rmhttp.New()
	app.Get("/test", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(reuseTestBody)
	})
	app.Compile()

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	// Warm the writer supply and any lazily initialized request state, then force a GC so the
	// measured iterations start from steady state.
	w := newReuseWriter()
	app.Router.ServeHTTP(w, req)
	w.release()
	runtime.GC()

	allocs := testing.AllocsPerRun(100, func() {
		w := newReuseWriter()
		app.Router.ServeHTTP(w, req)
		w.release()
	})

	t.Logf("measured allocations per acquire/serve/release cycle: %.0f", allocs)
	assert.LessOrEqual(t, allocs, 0.0,
		"the reusable writer cycle must not allocate; harness allocations pollute every benchmark")
}
