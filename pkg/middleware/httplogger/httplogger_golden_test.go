package httplogger

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------------------------------------------
// GOLDEN LOG CONTRACT
//
// The JSON shape the middleware emits is a public contract consumed by log pipelines, so it is
// pinned here byte-for-byte: field names, field order, values, and the level/msg pairing. The two
// volatile fields are matched by pattern instead of literally: the record timestamp, and the
// request duration in milliseconds (0 for a handler this fast, but it is legitimately non-deterministic).
//
// These expectations were captured from the middleware before any performance work and must stay
// byte-identical afterwards.
// ------------------------------------------------------------------------------------------------

const (
	timestampPattern = `"time":"RFC3339NANO"`
	durationPattern  = `"duration":DURATIONMS`
)

// goldenMatcher turns a frozen JSON log line into an anchored regexp that matches exactly that line,
// allowing for the volatile timestamp and duration values.
func goldenMatcher(t *testing.T, golden string) *regexp.Regexp {
	t.Helper()

	pattern := regexp.QuoteMeta(golden)
	pattern = strings.ReplaceAll(pattern, regexp.QuoteMeta(timestampPattern), `"time":"[^"]*"`)
	pattern = strings.ReplaceAll(pattern, regexp.QuoteMeta(durationPattern), `"duration":-?[0-9]+`)

	return regexp.MustCompile("^" + pattern + "\n$")
}

// Test_Middleware_GoldenLogOutput freezes the complete emitted field set for both log branches.
func Test_Middleware_GoldenLogOutput(t *testing.T) {
	newGoldenRequest := func() *http.Request {
		req := newBenchRequest("203.0.113.7:52000", []string{"203.0.113.7, 198.51.100.9"}, "")
		req.URL.Path = "/api/v1/users"
		req.URL.RawQuery = "limit=10&offset=20"
		req.Header.Set("Referer", "https://example.com/from")
		req.Header.Set("User-Agent", "rmhttp-test-agent/1.0")
		req.Host = "api.example.com"

		return req
	}

	tests := []struct {
		name    string
		status  int
		body    string
		golden  string
		request func() *http.Request
	}{
		{
			name:   "success branch logs INFO with the implicit 200 status",
			status: http.StatusOK,
			body:   "body",
			golden: `{"time":"RFC3339NANO","level":"INFO","msg":"OK","type":"http","status":200,` +
				`"ip":"203.0.113.7","method":"GET","host":"api.example.com",` +
				`"path":"/api/v1/users?limit=10&offset=20","referer":"https://example.com/from",` +
				`"ua":"rmhttp-test-agent/1.0","proto":"HTTP/1.1","size":4,"duration":DURATIONMS}`,
			request: newGoldenRequest,
		},
		{
			name:   "explicit created status keeps its own message",
			status: http.StatusCreated,
			body:   "created",
			golden: `{"time":"RFC3339NANO","level":"INFO","msg":"Created","type":"http","status":201,` +
				`"ip":"203.0.113.7","method":"GET","host":"api.example.com",` +
				`"path":"/api/v1/users?limit=10&offset=20","referer":"https://example.com/from",` +
				`"ua":"rmhttp-test-agent/1.0","proto":"HTTP/1.1","size":7,"duration":DURATIONMS}`,
			request: newGoldenRequest,
		},
		{
			name:   "client error branch logs ERROR",
			status: http.StatusNotFound,
			body:   "missing",
			golden: `{"time":"RFC3339NANO","level":"ERROR","msg":"Not Found","type":"http","status":404,` +
				`"ip":"203.0.113.7","method":"GET","host":"api.example.com",` +
				`"path":"/api/v1/users?limit=10&offset=20","referer":"https://example.com/from",` +
				`"ua":"rmhttp-test-agent/1.0","proto":"HTTP/1.1","size":7,"duration":DURATIONMS}`,
			request: newGoldenRequest,
		},
		{
			name:   "server error branch logs ERROR",
			status: http.StatusInternalServerError,
			body:   "boom",
			golden: `{"time":"RFC3339NANO","level":"ERROR","msg":"Internal Server Error","type":"http","status":500,` +
				`"ip":"203.0.113.7","method":"GET","host":"api.example.com",` +
				`"path":"/api/v1/users?limit=10&offset=20","referer":"https://example.com/from",` +
				`"ua":"rmhttp-test-agent/1.0","proto":"HTTP/1.1","size":4,"duration":DURATIONMS}`,
			request: newGoldenRequest,
		},
		{
			name:   "X-Forwarded-Host replaces the request host",
			status: http.StatusNoContent,
			body:   "",
			golden: `{"time":"RFC3339NANO","level":"INFO","msg":"No Content","type":"http","status":204,` +
				`"ip":"203.0.113.7","method":"GET","host":"proxied.example.com",` +
				`"path":"/api/v1/users?limit=10&offset=20","referer":"https://example.com/from",` +
				`"ua":"rmhttp-test-agent/1.0","proto":"HTTP/1.1","size":0,"duration":DURATIONMS}`,
			request: func() *http.Request {
				req := newGoldenRequest()
				req.Header.Set("X-Forwarded-Host", "proxied.example.com")

				return req
			},
		},
		{
			name:   "control characters are stripped from every untrusted field",
			status: http.StatusOK,
			body:   "body",
			golden: `{"time":"RFC3339NANO","level":"INFO","msg":"OK","type":"http","status":200,` +
				`"ip":"203.0.113.7","method":"GET","host":"evil.example.comforged",` +
				`"path":"/api/v1/users?limit=10&offset=20","referer":"https://example.com/forgedforged",` +
				`"ua":"evil-agentforged line","proto":"HTTP/1.1","size":4,"duration":DURATIONMS}`,
			request: func() *http.Request {
				req := newGoldenRequest()
				req.Header.Set("Referer", "https://example.com/forged\nforged")
				req.Header.Set("User-Agent", "evil-agent\nforged line")
				req.Header.Set("X-Forwarded-Host", "evil.example.com\nforged")

				return req
			},
		},
		{
			name:   "an unresolvable client IP degrades to an empty field",
			status: http.StatusOK,
			body:   "body",
			golden: `{"time":"RFC3339NANO","level":"INFO","msg":"OK","type":"http","status":200,` +
				`"ip":"","method":"GET","host":"api.example.com",` +
				`"path":"/api/v1/users?limit=10&offset=20","referer":"https://example.com/from",` +
				`"ua":"rmhttp-test-agent/1.0","proto":"HTTP/1.1","size":4,"duration":DURATIONMS}`,
			request: func() *http.Request {
				req := newGoldenRequest()
				req.RemoteAddr = ""
				req.Header.Del("X-Forwarded-For")

				return req
			},
		},
		{
			name:   "a path without a query is logged without the question mark",
			status: http.StatusOK,
			body:   "body",
			golden: `{"time":"RFC3339NANO","level":"INFO","msg":"OK","type":"http","status":200,` +
				`"ip":"203.0.113.7","method":"GET","host":"api.example.com",` +
				`"path":"/api/v1/users","referer":"https://example.com/from",` +
				`"ua":"rmhttp-test-agent/1.0","proto":"HTTP/1.1","size":4,"duration":DURATIONMS}`,
			request: func() *http.Request {
				req := newGoldenRequest()
				req.URL.RawQuery = ""

				return req
			},
		},
		{
			name:   "percent-escaped paths are logged in escaped form",
			status: http.StatusOK,
			body:   "body",
			golden: `{"time":"RFC3339NANO","level":"INFO","msg":"OK","type":"http","status":200,` +
				`"ip":"203.0.113.7","method":"GET","host":"api.example.com",` +
				`"path":"/api/v1/users/a%20b","referer":"https://example.com/from",` +
				`"ua":"rmhttp-test-agent/1.0","proto":"HTTP/1.1","size":4,"duration":DURATIONMS}`,
			request: func() *http.Request {
				req := newGoldenRequest()
				req.URL.Path = "/api/v1/users/a b"
				req.URL.RawPath = "/api/v1/users/a%20b"
				req.URL.RawQuery = ""

				return req
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := Middleware()(http.HandlerFunc(createTestHandlerFunc(test.status, test.body)))

			out.Reset()
			handler.ServeHTTP(httptest.NewRecorder(), test.request())

			assert.Regexp(t, goldenMatcher(t, test.golden), out.String(),
				"the emitted log line must stay byte-identical to the frozen contract")
		})
	}
}

// Test_Middleware_DefaultLoggerPerRequest pins that the logger is resolved from slog.Default() on
// every request instead of being captured when the middleware is built. Every test suite in this
// repository retargets the global logger, so a construction-time snapshot would silently send logs
// to the pre-test destination.
func Test_Middleware_DefaultLoggerPerRequest(t *testing.T) {
	t.Cleanup(func() { slog.SetDefault(slog.New(slog.NewJSONHandler(out, nil))) })

	handler := Middleware()(http.HandlerFunc(createTestHandlerFunc(http.StatusOK, "body")))
	req := newBenchRequest("203.0.113.7:52000", nil, "")

	out.Reset()
	handler.ServeHTTP(httptest.NewRecorder(), req)
	require.Contains(
		t,
		out.String(),
		`"ip":"203.0.113.7"`,
		"first request must land in the default logger",
	)

	var redirected bytes.Buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(&redirected, nil)))

	out.Reset()
	handler.ServeHTTP(httptest.NewRecorder(), req)

	assert.Empty(
		t,
		out.String(),
		"a rebuilt default logger must not keep writing to the previous destination",
	)
	assert.Contains(t, redirected.String(), `"ip":"203.0.113.7"`,
		"the second request must follow the logger that is default at the time it is handled")
}
