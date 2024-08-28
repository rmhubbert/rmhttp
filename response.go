package rmhttp

import (
	"bufio"
	"net"
	"net/http"
	"sync"
)

// ------------------------------------------------------------------------------------------------
// CAPTURE WRITER
// ------------------------------------------------------------------------------------------------

// A captureWriter wraps a http.ResponseWriter in order to capture HTTP the response code, body &
// headers that handlers will set. We do this to allow further processing based on this values
// before the final response is written, as writing a response status code can only be done
// once.
type captureWriter struct {
	writer        http.ResponseWriter
	code          int
	body          string
	header        http.Header
	mu            sync.Mutex
	headerWritten bool
}

func newCaptureWriter(w http.ResponseWriter) *captureWriter {
	return &captureWriter{
		writer: w,
		header: make(http.Header),
	}
}

// Write implements part of the http.ResponseWriter interface. We override it here in order to
// store the response body without actually writing the response.
func (cw *captureWriter) Write(body []byte) (int, error) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	if !cw.headerWritten {
		cw.WriteHeader(http.StatusOK)
	}
	cw.body = string(body)
	return len(cw.body), nil
}

// WriteHeader implements part of the http.ResponseWriter interface. We override it here in order to
// store the response code without actually writing the response.
func (cw *captureWriter) WriteHeader(code int) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	// We're only interested in storing the first header write, as the actual writer won't allow
	// multiple writes.
	if !cw.headerWritten {
		cw.code = code
		cw.headerWritten = true
	}
}

// Header implements part of the http.ResponseWriter interface. We override it here in order to
// store any added headers without actually writing the response.
func (cw *captureWriter) Header() http.Header { return cw.header }

// Push implements the Pusher interface.
func (cw *captureWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := cw.writer.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// Hijack implements the Hijacker interface.
func (cw *captureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := cw.writer.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return &net.TCPConn{}, bufio.NewReadWriter(
		bufio.NewReader(&bufio.Reader{}),
		bufio.NewWriter(&bufio.Writer{}),
	), http.ErrNotSupported
}

// Flush implements the Flusher interface.
func (cw *captureWriter) Flush() {
	if flusher, ok := cw.writer.(http.Flusher); ok {
		flusher.Flush()
	}
}
