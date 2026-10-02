package recoverer

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// RECOVERER TESTS
// ------------------------------------------------------------------------------------------------

const (
	testAddress string = "localhost:8123"
)

// captureLogs redirects the default logger into a buffer for the duration of the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	buf := &bytes.Buffer{}
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	return buf
}

// headerTracker records whether the handler committed a response status.
type headerTracker struct {
	http.ResponseWriter
	committed bool
}

func (w *headerTracker) WriteHeader(status int) {
	w.committed = true
	w.ResponseWriter.WriteHeader(status)
}

// testRequest builds the request shape the existing tests use.
func testRequest(t *testing.T, method, path string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, fmt.Sprintf("http://%s%s", testAddress, path), nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	return req
}

// Test_Recoverer checks that a panic thrown within a request is recovered, logged with
// its value and stack, and answered with a 500.
func Test_Recoverer(t *testing.T) {
	tests := []struct {
		name         string
		wantStatus   int
		wantLogged   bool
		setUpHeaders func(*http.Request)
		handler      http.Handler
	}{
		{
			"a panic is logged and answered with a 500",
			http.StatusInternalServerError,
			true,
			nil,
			http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				panic("Panicked!")
			}),
		},
		{
			"an upgrade request is logged but gets no status write",
			http.StatusOK,
			true,
			func(r *http.Request) { r.Header.Set("Connection", "Upgrade") },
			http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				panic("mid-tunnel failure")
			}),
		},
		{
			"nothing is logged or altered when no panic occurs",
			http.StatusOK,
			false,
			nil,
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logs := captureLogs(t)

			recorder := httptest.NewRecorder()
			tracker := &headerTracker{ResponseWriter: recorder}
			req := testRequest(t, http.MethodGet, "/test")
			if test.setUpHeaders != nil {
				test.setUpHeaders(req)
			}

			Middleware()(test.handler).ServeHTTP(tracker, req)

			assert.Equal(t, test.wantStatus, recorder.Code, "they should be equal")

			if test.wantLogged {
				output := logs.String()
				assert.Contains(t, output, "panic recovered", "the panic must be logged")
				assert.Contains(t, output, "/test", "the request path must be logged")
				assert.Contains(t, output, "Test_Recoverer", "the panic stack must be logged")
			} else {
				assert.Empty(t, logs.String(), "no log output is expected")
			}
		})
	}
}

// Test_Recoverer_AbortHandler checks that http.ErrAbortHandler is returned untouched to
// net/http — re-raised, unanswered and unlogged — because only the server can act on it.
func Test_Recoverer_AbortHandler(t *testing.T) {
	logs := captureLogs(t)

	recorder := httptest.NewRecorder()
	tracker := &headerTracker{ResponseWriter: recorder}

	abort := func() (recovered any) {
		defer func() { recovered = recover() }()
		Middleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic(http.ErrAbortHandler)
		})).ServeHTTP(tracker, testRequest(t, http.MethodGet, "/sse"))
		return nil
	}()

	assert.Equal(
		t,
		http.ErrAbortHandler,
		abort,
		"the sentinel must reach net/http's own recovery",
	)
	assert.False(t, tracker.committed, "no status may be written before the connection drops")
	assert.Empty(t, logs.String(), "the quiet abort signal must not be logged")
}

// Test_Recoverer_PanicNil checks the Go 1.21+ shape of panic(nil): the runtime converts it
// to a *runtime.PanicNilError, so it must be logged and answered, not treated as no panic.
func Test_Recoverer_PanicNil(t *testing.T) {
	logs := captureLogs(t)

	recorder := httptest.NewRecorder()
	tracker := &headerTracker{ResponseWriter: recorder}

	Middleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(nil)
	})).ServeHTTP(tracker, testRequest(t, http.MethodGet, "/nil"))

	assert.True(t, tracker.committed, "a nil panic must still produce a status")
	assert.Equal(
		t,
		http.StatusInternalServerError,
		recorder.Code,
		"they should be equal",
	)
	assert.Contains(t, logs.String(), "nil argument", "the runtime error text must be logged")
}

// Test_Recoverer_LogsAreSanitized checks that client-controlled text cannot inject extra
// log records: newlines in the path or the panic value must not survive into the output.
func Test_Recoverer_LogsAreSanitized(t *testing.T) {
	logs := captureLogs(t)

	Middleware()(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom\nSECOND RECORD")
	})).ServeHTTP(
		&headerTracker{ResponseWriter: httptest.NewRecorder()},
		testRequest(t, http.MethodGet, "/ok%0Ainjected"),
	)

	output := logs.String()
	assert.Equal(t, 1, strings.Count(output, "\n"), "one sanitized record, no injected lines")
	assert.NotContains(t, output, "SECOND RECORD\n", "panic text must not carry raw newlines")
	assert.NotContains(t, output, "injected\n", "path text must not carry raw newlines")
}
