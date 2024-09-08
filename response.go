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
	Writer http.ResponseWriter
	Code   int
	Body   string
	header http.Header
	Mu     sync.Mutex
}

func NewCaptureWriter(w http.ResponseWriter) *captureWriter {
	return &captureWriter{
		Writer: w,
		header: make(http.Header),
	}
}

// Write implements part of the http.ResponseWriter interface. We override it here in order to
// store the response body without actually writing the response.
func (cw *captureWriter) Write(body []byte) (int, error) {
	cw.Mu.Lock()
	defer cw.Mu.Unlock()
	cw.Body = string(body)
	return len(cw.Body), nil
}

// WriteHeader implements part of the http.ResponseWriter interface. We override it here in order to
// store the response code without actually writing the response.
func (cw *captureWriter) WriteHeader(code int) {
	cw.Mu.Lock()
	defer cw.Mu.Unlock()
	cw.Code = code
}

// Header implements part of the http.ResponseWriter interface. We override it here in order to
// store any added headers without actually writing the response.
func (cw *captureWriter) Header() http.Header { return cw.header }

// Persist writes the current status, body and headers to the underlying ResponseWriter.
func (cw *captureWriter) Persist() {
	cw.Writer.WriteHeader(cw.Code)
	_, _ = cw.Writer.Write([]byte(cw.Body))

	header := cw.Writer.Header()
	for key, value := range cw.header {
		header[key] = value
	}
}

// Push implements the Pusher interface.
func (cw *captureWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := cw.Writer.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// Hijack implements the Hijacker interface.
func (cw *captureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := cw.Writer.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return &net.TCPConn{}, bufio.NewReadWriter(
		bufio.NewReader(&bufio.Reader{}),
		bufio.NewWriter(&bufio.Writer{}),
	), http.ErrNotSupported
}

// Flush implements the Flusher interface.
func (cw *captureWriter) Flush() {
	if flusher, ok := cw.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}
