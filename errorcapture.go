package rmhttp

import (
	"bytes"
	"net/http"
	"sync"
)

// ------------------------------------------------------------------------------------------------
// ERROR CAPTURE
//
// The Router has to know whether the mux's internal 404/405 handler answered (and with what
// status) before deciding between a custom error handler and the stdlib response. The handler's
// output is therefore produced exactly once into the pooled errorCapture below, and is then
// either replaced by the custom handler or replayed onto the real writer — the mux's error
// handler never executes twice per request.
//
// Unlike CaptureWriter, an errorCapture always buffers the status and body and never passes the
// response through, so a replay can reproduce it byte-for-byte. Headers are the exception:
// Header() delegates to the real writer, so a handler's header writes land on the real response
// as they happen and need no replay.
// ------------------------------------------------------------------------------------------------

// errorCapturePool provides per-request errorCapture instances to keep the error path free of
// steady-state allocations. The body buffer keeps its capacity between requests.
var errorCapturePool = sync.Pool{
	New: func() any {
		return &errorCapture{}
	},
}

// An errorCapture records the status and body of one handler execution without passing any
// response through to an underlying ResponseWriter. Header() still delegates to that writer: a
// handler's header writes (the Allow header on a 405, for example) belong on the real response
// exactly as they would without the capture, so they land there directly and only the pieces a
// replay needs — status and body — are buffered. It is designed for a single request/response
// cycle and is not safe for concurrent use.
type errorCapture struct {
	writer http.ResponseWriter
	status int
	body   bytes.Buffer
}

func newErrorCapture(w http.ResponseWriter) *errorCapture {
	ec := errorCapturePool.Get().(*errorCapture)
	ec.writer = w
	ec.status = http.StatusOK
	return ec
}

// release resets the capture and returns it to the pool. The caller must have finished with the
// captured status and body bytes before calling it.
func (ec *errorCapture) release() {
	ec.writer = nil
	ec.body.Reset()
	errorCapturePool.Put(ec)
}

func (ec *errorCapture) Header() http.Header { return ec.writer.Header() }

func (ec *errorCapture) Write(p []byte) (int, error) { return ec.body.Write(p) }

func (ec *errorCapture) WriteHeader(status int) { ec.status = status }
