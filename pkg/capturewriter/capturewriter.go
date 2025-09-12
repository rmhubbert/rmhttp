package capturewriter

import (
	"bufio"
	"net"
	"net/http"
	"sync"
)

// ------------------------------------------------------------------------------------------------
// CAPTURE WRITER
// ------------------------------------------------------------------------------------------------

// A CaptureWriter wraps a http.ResponseWriter in order to capture HTTP the response code, body &
// headers that handlers will set. We do this to allow further processing based on this values
// before the final response is written, as writing a response status code can only be done
// once.
type CaptureWriter struct {
	Writer http.ResponseWriter
	Code   int
	Body   string
	header http.Header
	Mu     sync.Mutex
}

func New(w http.ResponseWriter) *CaptureWriter {
	return &CaptureWriter{
		Writer: w,
		header: make(http.Header),
	}
}

// Write implements part of the http.ResponseWriter interface. We override it here in order to
// store the response body without actually writing the response.
func (cw *CaptureWriter) Write(body []byte) (int, error) {
	cw.Mu.Lock()
	defer cw.Mu.Unlock()
	cw.Body = string(body)
	return len(cw.Body), nil
}

// WriteHeader implements part of the http.ResponseWriter interface. We override it here in order to
// store the response code without actually writing the response.
func (cw *CaptureWriter) WriteHeader(code int) {
	cw.Mu.Lock()
	defer cw.Mu.Unlock()
	cw.Code = code
}

// Header implements part of the http.ResponseWriter interface. We override it here in order to
// store any added headers without actually writing the response.
func (cw *CaptureWriter) Header() http.Header { return cw.header }

// Persist writes the current status, body and headers to the underlying ResponseWriter.
func (cw *CaptureWriter) Persist() {
	cw.Mu.Lock()
	defer cw.Mu.Unlock()

	// Order is important here. WriteHeader writes all headers, not just the status code, so we need
	// to add any other headers before calling WriteHeader.
	header := cw.Writer.Header()
	for key, values := range cw.header {
		for _, value := range values {
			header.Add(key, value)
		}
	}
	// Also, it's important that we call WriteHeader before Write, as Write will call WriteHeader with
	// a 200 status code, if it hasn't already been set.
	cw.Writer.WriteHeader(cw.Code)
	_, _ = cw.Writer.Write([]byte(cw.Body))
}

// Push implements the Pusher interface.
func (cw *CaptureWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := cw.Writer.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// Hijack implements the Hijacker interface.
func (cw *CaptureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := cw.Writer.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return &net.TCPConn{}, bufio.NewReadWriter(
		bufio.NewReader(&bufio.Reader{}),
		bufio.NewWriter(&bufio.Writer{}),
	), http.ErrNotSupported
}

// Flush implements the Flusher interface.
func (cw *CaptureWriter) Flush() {
	if flusher, ok := cw.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}
