package httplogger

import (
	"log/slog"
	"net/http"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test_Middleware_AllocBudget_LoggerDisabled (Phase 5 red/green gate): when the default logger
// discards the record's level, the middleware must skip the per-request cost of client-IP
// resolution, sanitization, the path+"?"+query concatenation, and attribute building. The budget
// is the one unavoidable allocation: the metrics-writer wrapper (which is never pooled, by
// design). At baseline the middleware pays for that work before handing the finished record to
// slog, so the concatenation of a query-bearing path already fails this gate.
func Test_Middleware_AllocBudget_LoggerDisabled(t *testing.T) {
	const budget = 1 // the response-writer wrapper, the only wrapper-essential allocation

	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	// A logger whose minimum level discards everything the middleware emits, including errors.
	// Restored afterwards so the package's other tests keep the TestMain default.
	t.Cleanup(func() { slog.SetDefault(slog.New(slog.NewJSONHandler(out, nil))) })
	slog.SetDefault(slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{
		Level: slog.Level(100),
	})))

	handler := Middleware()(allocTestHandler(http.StatusOK))

	req := newBenchRequest("203.0.113.7:52000", []string{"203.0.113.7, 198.51.100.9"}, "")
	req.URL.Path = "/api/v1/users"
	req.URL.RawQuery = "limit=10&offset=20"
	req.Header.Set("Referer", "https://example.com/from")
	req.Header.Set("User-Agent", "rmhttp-test-agent/1.0")
	req.Host = "api.example.com"

	w := newReuseWriter()
	handler.ServeHTTP(w, req) // warm pools and lazily initialized state
	runtime.GC()

	allocs := testing.AllocsPerRun(200, func() {
		out.Reset()
		w.reset()
		handler.ServeHTTP(w, req)
	})

	t.Logf("measured allocations per discarded request: %.0f (budget %d)", allocs, budget)
	assert.LessOrEqual(t, allocs, float64(budget),
		"a discarded log line must not pay for field extraction, sanitization, or concatenation")
}
