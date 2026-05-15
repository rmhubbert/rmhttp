package rmhttp

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"sync"
)

// ------------------------------------------------------------------------------------------------
// CAPTURE WRITER
// ------------------------------------------------------------------------------------------------

// captureWriterPool provides a pool of CaptureWriter instances to reduce per-request allocations.
var captureWriterPool = sync.Pool{
	New: func() any {
		return &CaptureWriter{
			Code:        http.StatusOK,
			PassThrough: true,
		}
	},
}

// A CaptureWriter wraps a http.ResponseWriter in order to capture the HTTP response code & body
// that handlers will set. This allows reading these values after they have been set,
// as this is not normally possible on a ResponseWriter.
//
// CaptureWriter is designed for a single request/response cycle. It is not safe for concurrent use.
type CaptureWriter struct {
	Writer      http.ResponseWriter
	Code        int
	buf         *bytes.Buffer
	PassThrough bool
}

// NewCaptureWriter creates, instantiates, and returns a new CaptureWriter.
func NewCaptureWriter(w http.ResponseWriter) *CaptureWriter {
	cw := captureWriterPool.Get().(*CaptureWriter)
	cw.Writer = w
	cw.Code = http.StatusOK
	cw.PassThrough = true
	cw.buf = nil
	return cw
}

// Body returns the captured response body as a string.
// It performs a single O(n) conversion, so call it only once after the handler completes.
// This is a breaking change from the previous Body field — the method enforces lazy evaluation
// to avoid O(n²) string conversions on every Write call.
func (cw *CaptureWriter) Body() string {
	if cw.buf == nil {
		return ""
	}
	return cw.buf.String()
}

// Write implements part of the http.ResponseWriter interface. It overrides the underlying Write
// to capture the response body, before optionally writing to the underlying ResponseWriter.
func (cw *CaptureWriter) Write(body []byte) (int, error) {
	if cw.PassThrough {
		if cw.buf == nil {
			cw.buf = &bytes.Buffer{}
		}
		cw.buf.Write(body)
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
