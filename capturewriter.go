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

// A CaptureWriter wraps a http.ResponseWriter in order to capture HTTP the response code & body
// that handlers will set. We do this to allow reading these values after they have been set,
// as this is not normally possible on a ResponseWriter.
type CaptureWriter struct {
	Writer      http.ResponseWriter
	Code        int
	Body        string
	Mu          sync.Mutex
	PassThrough bool
}

// New creates, instantiates, and returns a new CaptureWriter.
func NewCaptureWriter(w http.ResponseWriter) *CaptureWriter {
	return &CaptureWriter{
		Writer:      w,
		PassThrough: true,
	}
}

// Write implements part of the http.ResponseWriter interface. We override it here in order to
// store the response body, before optionally writing to the underlying ResponseWriter.
func (cw *CaptureWriter) Write(body []byte) (int, error) {
	cw.Mu.Lock()
	defer cw.Mu.Unlock()
	cw.Body = string(body)
	if cw.PassThrough {
		return cw.Writer.Write(body)
	}
	return len(cw.Body), nil
}

// WriteHeader implements part of the http.ResponseWriter interface. We override it here in order to
// store the response code, before optionally writing to the underlying ResponseWriter.
func (cw *CaptureWriter) WriteHeader(code int) {
	cw.Mu.Lock()
	defer cw.Mu.Unlock()
	cw.Code = code
	if cw.PassThrough {
		cw.Writer.WriteHeader(code)
	}
}

// Header implements part of the http.ResponseWriter interface. We simply pass this to the
// underlying ResponseWriter, as you can already retrieve a Header from that.
func (cw *CaptureWriter) Header() http.Header {
	return cw.Writer.Header()
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

// Unwrap returns the underlying http.ResponseWriter. It is used internally by
// http.ResponseController, which allows you to use custom http.ResponseWriter
// instances more easily.
func (cw *CaptureWriter) Unwrap() http.ResponseWriter {
	return cw.Writer
}
