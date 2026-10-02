package benchmarks

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
)

// ------------------------------------------------------------------------------------------------
// SHARED HOT-PATH RESPONSE WRITER
//
// reuseWriter is the test package's reusable http.ResponseWriter, modeled on the proven test
// double in pkg/middleware/httplogger. Creating a fresh httptest.ResponseRecorder inside a timing
// loop costs more allocations than the router path under measurement (the baseline profile showed
// ~83% of benchmark alloc bytes coming from recorder internals), so hot-path benchmarks acquire
// one writer outside the loop and recycle it with reset().
//
// It advertises the same optional interfaces as a net/http HTTP/1 server writer (Flusher,
// Hijacker, io.ReaderFrom) plus http.Pusher, so middleware such as httplogger selects the same
// writer-wrapping path it would in production.
// ------------------------------------------------------------------------------------------------

type reuseWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

// reuseWriterSupply is a channel-based pool. Unlike sync.Pool it is not emptied by runtime.GC,
// which keeps the Test_ReuseWriter_ZeroAllocs assertion deterministic even though
// testing.AllocsPerRun performs GC cycles internally.
var reuseWriterSupply = make(chan *reuseWriter, 64)

func newReuseWriter() *reuseWriter {
	select {
	case w := <-reuseWriterSupply:
		return w
	default:
		return &reuseWriter{header: make(http.Header, 8), status: http.StatusOK}
	}
}

// reset clears the captured response, reusing all existing storage so the next request allocates
// nothing in this writer. The writer stays owned by the caller (see release).
func (w *reuseWriter) reset() {
	w.status = http.StatusOK
	w.body.Reset()
	clear(w.header)
}

// release returns the writer to the supply; if the supply is full it is simply dropped.
func (w *reuseWriter) release() {
	select {
	case reuseWriterSupply <- w:
	default:
	}
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
