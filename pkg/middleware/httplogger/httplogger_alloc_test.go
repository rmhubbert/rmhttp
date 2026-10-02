package httplogger

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// allocTestBody is a pre-allocated body: writing it costs the handler nothing, so the measurements
// below count the middleware's allocations and not those of the test fixture. createTestHandlerFunc
// cannot be used here because its []byte(string) conversion adds an allocation the middleware does
// not own.
var allocTestBody = []byte("body")

// allocTestHandler returns a handler that writes a fixed status and body without allocating.
func allocTestHandler(status int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write(allocTestBody)
	})
}

// Test_Middleware_AllocBudget pins the per-request allocation budget of the logging hot path.
//
// The middleware runs on every request, so its cost is multiplied by request volume. The budget is
// three allocations:
//
//  1. the response-writer wrapper, which must escape to the heap because it is passed to the next
//     handler through an interface (and must never be pooled, because streaming and hijacked
//     handlers can keep using it after ServeHTTP returns);
//  2. the slog record's overflow attribute slice, which slog allocates once a record carries more
//     than its five inline attributes;
//  3. the path+"?"+query concatenation, which only exists on requests that carry a query string.
//
// The ResponseWriter and the log sink are created outside the measured closure on purpose: they are
// not part of the middleware, and allocating a fresh httptest.ResponseRecorder per iteration would
// measure the test harness instead of the code under test.
func Test_Middleware_AllocBudget(t *testing.T) {
	const budget = 3

	tests := []struct {
		name    string
		status  int
		query   string
		xff     []string
		maxFrac float64
	}{
		{name: "success path", status: http.StatusOK, maxFrac: budget},
		{name: "error path", status: http.StatusInternalServerError, maxFrac: budget},
		{
			name:    "query string path",
			status:  http.StatusOK,
			query:   "limit=10&offset=20",
			maxFrac: budget,
		},
		{
			name:    "full header set path",
			status:  http.StatusCreated,
			query:   "limit=10",
			xff:     []string{"203.0.113.7, 198.51.100.9"},
			maxFrac: budget,
		},
	}

	if raceEnabled {
		t.Skip(
			"the race detector allocates on almost every access; run without -race to measure the budget",
		)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := Middleware()(allocTestHandler(test.status))

			req := newBenchRequest("203.0.113.7:52000", test.xff, "")
			req.Header.Set("Referer", "https://example.com/from")
			req.Header.Set("User-Agent", "rmhttp-test-agent/1.0")
			req.Host = "api.example.com"
			if test.query != "" {
				req.URL.RawQuery = test.query
			}

			w := newReuseWriter()
			allocs := testing.AllocsPerRun(200, func() {
				out.Reset()
				w.reset()
				handler.ServeHTTP(w, req)
			})

			t.Logf("measured allocations per request: %.0f (budget %.0f)", allocs, test.maxFrac)
			assert.LessOrEqual(t, allocs, float64(test.maxFrac),
				"middleware must not exceed its per-request allocation budget")
		})
	}
}
