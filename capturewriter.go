package rmhttp

import (
	"bufio"
	"net"
	"net/http"
)

// ------------------------------------------------------------------------------------------------
// CAPTURE WRITER
// ------------------------------------------------------------------------------------------------

// A CaptureWriter wraps a http.ResponseWriter in order to capture the HTTP response code & body
// that handlers will set. This allows reading these values after they have been set,
// as this is not normally possible on a ResponseWriter.
//
// CaptureWriter is designed for a single request/response cycle. It is not safe for concurrent use.
type CaptureWriter struct {
	Writer      http.ResponseWriter
	Code        int
	Body        string
	bodyAcc     []byte
	PassThrough bool
}

// NewCaptureWriter creates, instantiates, and returns a new CaptureWriter.
func NewCaptureWriter(w http.ResponseWriter) *CaptureWriter {
	return &CaptureWriter{
		Writer:      w,
		Code:        http.StatusOK,
		PassThrough: true,
		bodyAcc:     make([]byte, 0, 1024), // Pre-allocate 1KB buffer to reduce reallocations
	}
}

// Write implements part of the http.ResponseWriter interface. It overrides the underlying Write
// to capture the response body, before optionally writing to the underlying ResponseWriter.
func (cw *CaptureWriter) Write(body []byte) (int, error) {
	if cw.PassThrough {
		cw.bodyAcc = append(cw.bodyAcc, body...)
		cw.Body = string(cw.bodyAcc)
		return cw.Writer.Write(body)
	}
	// When PassThrough is false, skip body capture and pass-through entirely.
	return len(body), nil
}

// WriteHeader implements part of the http.ResponseWriter interface. It overrides the underlying
// WriteHeader to capture the response code, before optionally writing to the underlying ResponseWriter.
func (cw *CaptureWriter) WriteHeader(code int) {
	cw.Code = code
	if cw.PassThrough {
		cw.Writer.WriteHeader(code)
	}
}

// Header implements part of the http.ResponseWriter interface. It simply delegates to the
// underlying ResponseWriter, since headers can already be accessed from that.
func (cw *CaptureWriter) Header() http.Header {
	return cw.Writer.Header()
}

// Push implements the http.Pusher interface.
func (cw *CaptureWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := cw.Writer.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// Hijack implements the http.Hijacker interface.
func (cw *CaptureWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := cw.Writer.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Flush implements the http.Flusher interface.
func (cw *CaptureWriter) Flush() {
	if flusher, ok := cw.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Unwrap returns the underlying http.ResponseWriter. It is used internally by
// http.ResponseController, which allows custom http.ResponseWriter
// instances to work more easily with the standard library.
func (cw *CaptureWriter) Unwrap() http.ResponseWriter {
	return cw.Writer
}
