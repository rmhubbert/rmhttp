package httplogger

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

// benchIPSink keeps resolver results observable to the compiler without
// adding allocations inside the timed loop.
var benchIPSink string

// newBenchRequest builds a request with a controlled RemoteAddr and
// X-Forwarded-For / X-Real-Ip headers. Requests are built OUTSIDE the timed
// loops so only the resolver call itself is measured.
func newBenchRequest(remoteAddr string, xffLines []string, realIP string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = remoteAddr
	for _, line := range xffLines {
		req.Header.Add("X-Forwarded-For", line)
	}
	if realIP != "" {
		req.Header.Set("X-Real-Ip", realIP)
	}
	return req
}

// Benchmark_realIp measures the per-request client IP resolution hot path.
// The function name and sub-benchmark names are the benchstat contract and
// are intentionally kept byte-identical to the Phase-A baseline capture of
// the original realIp function.
func Benchmark_realIp(b *testing.B) {
	b.ReportAllocs()

	cases := []struct {
		name       string
		remoteAddr string
		xffLines   []string
		realIP     string
	}{
		{"NoHeaders", "203.0.113.7:52000", nil, ""},
		{"XFF_Single", "10.0.0.1:80", []string{"203.0.113.7"}, ""},
		{
			"XFF_Multi_4",
			"10.0.0.1:80",
			[]string{"203.0.113.7, 198.51.100.9, 192.0.2.24, 10.0.0.2"},
			"",
		},
		{"XRealIP", "10.0.0.1:80", nil, "203.0.113.7"},
		{"RemoteAddr_IPv6", "[2001:db8::1]:8080", nil, ""},
		{"XFF_Garbage", "10.0.0.1:80", []string{"not-an-ip, also-not-an-ip"}, ""},
		{"XFF_DupHeaderLines", "10.0.0.1:80", []string{"203.0.113.7", "198.51.100.9"}, ""},
	}

	for _, tc := range cases {
		req := newBenchRequest(tc.remoteAddr, tc.xffLines, tc.realIP)
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				benchIPSink = clientIP(req)
			}
		})
	}
}

// Benchmark_realIp_Parallel proves the resolver read path is contention-free
// under concurrent use (pure function, no global logger access).
func Benchmark_realIp_Parallel(b *testing.B) {
	b.ReportAllocs()

	req := newBenchRequest("203.0.113.7:52000", nil, "")
	b.RunParallel(func(pb *testing.PB) {
		var total int
		for pb.Next() {
			total += len(clientIP(req))
		}
		if total >= 0 {
			benchIPSink = req.RemoteAddr // single write per goroutine, keeps call live
		}
	})
}

// ------------------------------------------------------------------------------------------------
// SHARED HOT-PATH TEST DOUBLE
// ------------------------------------------------------------------------------------------------

// reuseWriter is a capability-rich http.ResponseWriter that can be reused across requests, so
// allocation measurements and benchmarks only count what the middleware itself allocates.
//
// It advertises the same optional interfaces as a net/http server writer (Flusher, Hijacker,
// io.ReaderFrom, http.Pusher) so the middleware selects the writer variant it would pick in
// production. httptest.ResponseRecorder is deliberately not used for this: it allocates on every
// request, which would swamp the measurement, and its capability set matches neither HTTP/1 nor
// HTTP/2 server writers.
type reuseWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newReuseWriter() *reuseWriter {
	return &reuseWriter{header: http.Header{}, status: http.StatusOK}
}

// reset clears the captured response, reusing all existing storage so the next request allocates
// nothing in this writer.
func (w *reuseWriter) reset() {
	w.status = http.StatusOK
	w.body.Truncate(0)
	clear(w.header)
}

func (w *reuseWriter) Header() http.Header { return w.header }

func (w *reuseWriter) WriteHeader(status int) { w.status = status }

func (w *reuseWriter) Write(p []byte) (int, error) { return w.body.Write(p) }

func (w *reuseWriter) Flush() {}

func (w *reuseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, http.ErrNotSupported
}

func (w *reuseWriter) ReadFrom(src io.Reader) (int64, error) { return io.Copy(&w.body, src) }

func (w *reuseWriter) Push(string, *http.PushOptions) error { return http.ErrNotSupported }

// ------------------------------------------------------------------------------------------------
// MIDDLEWARE HOT-PATH BENCHMARKS
//
// These measure the full per-request path: writer wrapping, handler execution, field extraction,
// sanitization and the slog call. The log sink is reset inside the loop so the shared buffer stays
// at steady-state capacity instead of growing forever, and the ResponseWriter is reused so the
// benchmark reports the middleware's own cost. Names are part of the benchstat contract: never
// rename or remove one. Run in series (no RunParallel) because the package log buffer is shared.
// ------------------------------------------------------------------------------------------------

// benchMiddlewareRequest builds a realistic request outside the timed loop.
func benchMiddlewareRequest(query string, xff []string) *http.Request {
	req := newBenchRequest("203.0.113.7:52000", xff, "")
	req.URL.Path = "/api/v1/users"
	req.URL.RawQuery = query
	req.Header.Set("Referer", "https://example.com/from")
	req.Header.Set("User-Agent", "rmhttp-test-agent/1.0")
	req.Host = "api.example.com"

	return req
}

func Benchmark_Middleware(b *testing.B) {
	b.ReportAllocs()

	handler := Middleware()(http.HandlerFunc(createTestHandlerFunc(http.StatusOK, "body")))
	req := benchMiddlewareRequest("", []string{"203.0.113.7, 198.51.100.9"})
	w := newReuseWriter()

	for b.Loop() {
		out.Reset()
		w.reset()
		handler.ServeHTTP(w, req)
	}

	benchIPSink = w.body.String()
}

func Benchmark_Middleware_Error(b *testing.B) {
	b.ReportAllocs()

	handler := Middleware()(
		http.HandlerFunc(createTestHandlerFunc(http.StatusInternalServerError, "body")),
	)
	req := benchMiddlewareRequest("", []string{"203.0.113.7, 198.51.100.9"})
	w := newReuseWriter()

	for b.Loop() {
		out.Reset()
		w.reset()
		handler.ServeHTTP(w, req)
	}

	benchIPSink = w.body.String()
}

func Benchmark_Middleware_Query(b *testing.B) {
	b.ReportAllocs()

	handler := Middleware()(http.HandlerFunc(createTestHandlerFunc(http.StatusOK, "body")))
	req := benchMiddlewareRequest("limit=10&offset=20", []string{"203.0.113.7, 198.51.100.9"})
	w := newReuseWriter()

	for b.Loop() {
		out.Reset()
		w.reset()
		handler.ServeHTTP(w, req)
	}

	benchIPSink = w.body.String()
}

// Benchmark_Middleware_Minimal is the floor: no proxy headers, no query, empty referer/UA, and the
// smallest possible body, so the difference to Benchmark_Middleware shows what the extra request
// metadata costs.
func Benchmark_Middleware_Minimal(b *testing.B) {
	b.ReportAllocs()

	handler := Middleware()(http.HandlerFunc(createTestHandlerFunc(http.StatusOK, "")))
	req := newBenchRequest("203.0.113.7:52000", nil, "")
	w := newReuseWriter()

	for b.Loop() {
		out.Reset()
		w.reset()
		handler.ServeHTTP(w, req)
	}

	benchIPSink = w.body.String()
}
