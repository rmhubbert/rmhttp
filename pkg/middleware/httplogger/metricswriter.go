package httplogger

import (
	"io"
	"net/http"
)

// ------------------------------------------------------------------------------------------------
// METRICS CAPTURE WRITER
//
// The middleware has to know the status code and the number of body bytes the handler produced,
// which a plain http.ResponseWriter cannot report afterwards. It therefore hands the next handler
// one of the wrapper types below.
//
// Why there is more than one wrapper type: an http.ResponseWriter may optionally implement
// http.Flusher, http.Hijacker, io.ReaderFrom or http.Pusher, and handlers discover those abilities by
// type-asserting the writer they are given. A single wrapper struct that declared Flush/Hijack/Push
// unconditionally would advertise capabilities the real writer may not have, and every failure mode
// of that mistake is silent:
//
//   - an SSE handler that asserts http.Flusher succeeds, its Flush() becomes a no-op, and events sit
//     in a buffer until the response ends instead of streaming;
//   - a websocket upgrader that asserts http.Hijacker succeeds and then panics on the nil connection;
//   - advertising io.ReaderFrom without a real implementation disables the kernel sendfile path.
//
// So each variant embeds only the capabilities the wrapped writer actually has, chosen once per
// request in newMetricsWriter. There is one variant for each of the sixteen combinations of the four
// capabilities, so a wrapper advertises exactly what the wrapped writer can do: never more, never
// less. The single deliberate exception is FlushError, which every flush-capable variant answers
// even when the base writer only has a bare Flush method — see the flushCore documentation. Unwrap
// keeps the real writer reachable for http.ResponseController either way.
//
// Why the wrappers are not pooled: handlers are allowed to keep using the ResponseWriter after
// ServeHTTP returns — the usual SSE pattern writes from another goroutine, and a hijacked connection
// hands the writer's connection to the caller for its lifetime. Recycling a wrapper that is still in
// use corrupts a different request, and sync.Pool disables the race detector's ability to see that
// use-after-Put (see the note in sync.Pool's own documentation and golang/go#17306). One allocation
// per request is the honest cost here.
// ------------------------------------------------------------------------------------------------

// statusRecorder is the capture core embedded by every writer variant. It records the response
// status and the number of body bytes written, and delegates everything else to the wrapped writer.
//
// statusRecorder itself is never handed to a handler; newMetricsWriter wraps it in a capability
// variant so the method set matches the underlying writer.
type statusRecorder struct {
	// ResponseWriter is the wrapped writer. Embedding it supplies Header(), and the optional
	// writer interfaces (Flusher, Hijacker, Pusher, io.ReaderFrom) are deliberately NOT promoted
	// from it: embedding an interface promotes only the methods that interface declares.
	http.ResponseWriter

	status      int
	written     int64
	wroteHeader bool

	// readerFrom is set only when the wrapped writer implements io.ReaderFrom, so the variants that
	// advertise ReadFrom can delegate without re-asserting.
	readerFrom io.ReaderFrom
}

// WriteHeader records the first status of 200 or above, mirroring how net/http commits a response
// status: 1xx informational responses pass through without committing, and any later call is a
// superfluous write that the server itself reports.
//
// Codes below 100 are deliberately not recorded. The previous implementation (httpsnoop) recorded
// the first code outside the 1xx range, so it would also record those, but no real server ever
// commits one: net/http's checkWriteHeaderCode panics on codes below 100 (and above 999) before a
// status is sent, and httptest.ResponseRecorder does the same. Logging such a code would report a
// status no client can receive, which is why the guard is ">= 200" and not "not 1xx". The call
// itself is always forwarded untouched, so the server still panics and reports exactly as it would
// without this wrapper.
func (s *statusRecorder) WriteHeader(status int) {
	if !s.wroteHeader && status >= http.StatusOK {
		s.status = status
		s.wroteHeader = true
	}

	s.ResponseWriter.WriteHeader(status)
}

// Write captures the body byte count and forwards the write.
func (s *statusRecorder) Write(p []byte) (int, error) {
	n, err := s.ResponseWriter.Write(p)
	s.countBody(n)

	return n, err
}

// WriteString keeps the string fast path that io.WriteString and text/template use. It is safe for a
// writer that has no WriteString method of its own: io.WriteString then does the byte-slice
// conversion while delegating to Write, which costs the same as it would without this method.
func (s *statusRecorder) WriteString(text string) (int, error) {
	n, err := io.WriteString(s.ResponseWriter, text)
	s.countBody(n)

	return n, err
}

// Unwrap returns the wrapped writer. http.ResponseController walks this chain, so deadline,
// full-duplex and other capabilities that these variants do not mirror stay reachable.
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

// countBody records n body bytes and commits the implicit 200 status on the first write.
func (s *statusRecorder) countBody(n int) {
	s.written += int64(n)
	s.wroteHeader = true
}

// countReadFrom is the ReadFrom implementation shared by the variants that advertise io.ReaderFrom.
// The kernel copy fast path must still be counted, or the logged size would be 0 for handlers that
// stream a file straight to the connection.
func (s *statusRecorder) countReadFrom(src io.Reader) (int64, error) {
	n, err := s.readerFrom.ReadFrom(src)
	s.written += n
	s.wroteHeader = true

	return n, err
}

// flushCore backs every variant that advertises http.Flusher. It holds the capability resolved once
// at wrap time, so flushing costs no further type assertions.
type flushCore struct {
	flusher http.Flusher

	// flushErr is nil when the wrapped writer cannot report flush errors.
	flushErr interface{ FlushError() error }
}

// Flush implements http.Flusher.
func (f flushCore) Flush() { f.flusher.Flush() }

// FlushError answers the capability probe that http.ResponseController.Flush prefers over a plain
// Flusher. When the wrapped writer can report flush errors they are returned; otherwise the flush
// happens here and nil is returned, which is what http.ResponseController itself does with a writer
// that only implements http.Flusher.
//
// This method is therefore intentionally advertised by every variant that embeds flushCore, even
// when the wrapped writer lacks it. The answer always matches the unwrapped writer, so this is the
// one place where the wrapper may advertise more than the base (the exception noted at the top of
// this file); do not "fix" it by only forwarding FlushError when the base implements it.
func (f flushCore) FlushError() error {
	if f.flushErr != nil {
		return f.flushErr.FlushError()
	}

	f.flusher.Flush()

	return nil
}

// ------------------------------------------------------------------------------------------------
// CAPABILITY VARIANTS
//
// Each variant is the capture core plus exactly the optional interfaces the wrapped writer supports.
// The name is the capability set: F = http.Flusher, H = http.Hijacker, R = io.ReaderFrom,
// P = http.Pusher, and basicWriter adds nothing. Flusher, Hijacker and Pusher require no code, since
// embedding an interface value promotes its single method; only the variants that advertise
// io.ReaderFrom declare ReadFrom, because bytes moved by the kernel copy path still have to be
// counted.
//
// In production two of these do the work: net/http's HTTP/1 writer lands on http1Writer and its
// HTTP/2 writer on flushPushWriter (both also expose FlushError, which flushCore forwards, plus
// deadlines and EnableFullDuplex, which stay reachable through Unwrap). httptest.ResponseRecorder
// lands on flushWriter and http.TimeoutHandler's writer on pushWriter; the rest exist so a writer
// from any other middleware is mirrored rather than guessed at.
// ------------------------------------------------------------------------------------------------

type fullWriter struct {
	statusRecorder
	flushCore
	http.Hijacker
	http.Pusher
}

func (w *fullWriter) ReadFrom(src io.Reader) (int64, error) { return w.countReadFrom(src) }

type http1Writer struct {
	statusRecorder
	flushCore
	http.Hijacker
}

func (w *http1Writer) ReadFrom(src io.Reader) (int64, error) { return w.countReadFrom(src) }

type flushHijackPushWriter struct {
	statusRecorder
	flushCore
	http.Hijacker
	http.Pusher
}

type flushReadPushWriter struct {
	statusRecorder
	flushCore
	http.Pusher
}

func (w *flushReadPushWriter) ReadFrom(src io.Reader) (int64, error) { return w.countReadFrom(src) }

type flushHijackWriter struct {
	statusRecorder
	flushCore
	http.Hijacker
}

type flushReadWriter struct {
	statusRecorder
	flushCore
}

func (w *flushReadWriter) ReadFrom(src io.Reader) (int64, error) { return w.countReadFrom(src) }

type flushPushWriter struct {
	statusRecorder
	flushCore
	http.Pusher
}

type flushWriter struct {
	statusRecorder
	flushCore
}

type hijackReadWriter struct {
	statusRecorder
	http.Hijacker
}

func (w *hijackReadWriter) ReadFrom(src io.Reader) (int64, error) { return w.countReadFrom(src) }

type hijackReadPushWriter struct {
	statusRecorder
	http.Hijacker
	http.Pusher
}

func (w *hijackReadPushWriter) ReadFrom(
	src io.Reader,
) (int64, error) {
	return w.countReadFrom(src)
}

type hijackPushWriter struct {
	statusRecorder
	http.Hijacker
	http.Pusher
}

type hijackWriter struct {
	statusRecorder
	http.Hijacker
}

type readPushWriter struct {
	statusRecorder
	http.Pusher
}

func (w *readPushWriter) ReadFrom(src io.Reader) (int64, error) { return w.countReadFrom(src) }

type readWriter struct {
	statusRecorder
}

func (w *readWriter) ReadFrom(src io.Reader) (int64, error) { return w.countReadFrom(src) }

type pushWriter struct {
	statusRecorder
	http.Pusher
}

type basicWriter struct {
	statusRecorder
}

// newMetricsWriter wraps w and returns the capture core together with the writer to hand to the next
// handler. The optional capabilities of w are resolved once, so the wrapper advertises a method only
// when the writer behind it really implements it.
//
// The wrapper is allocated per request and is never reused: see the note about pooling at the top of
// this file. It escapes to the heap because it is handed to the next handler through an interface,
// which is the single unavoidable allocation on this path.
func newMetricsWriter(w http.ResponseWriter) (*statusRecorder, http.ResponseWriter) {
	rec := statusRecorder{ResponseWriter: w, status: http.StatusOK}

	var (
		flush  flushCore
		hijack http.Hijacker
		push   http.Pusher
	)

	if f, ok := w.(http.Flusher); ok {
		flush.flusher = f
		if fe, ok := w.(interface{ FlushError() error }); ok {
			flush.flushErr = fe
		}
	}

	if h, ok := w.(http.Hijacker); ok {
		hijack = h
	}

	if rf, ok := w.(io.ReaderFrom); ok {
		rec.readerFrom = rf
	}

	if p, ok := w.(http.Pusher); ok {
		push = p
	}

	canFlush, canHijack := flush.flusher != nil, hijack != nil
	canRead, canPush := rec.readerFrom != nil, push != nil

	switch {
	case canFlush && canHijack && canRead && canPush:
		v := &fullWriter{statusRecorder: rec, flushCore: flush, Hijacker: hijack, Pusher: push}
		return &v.statusRecorder, v

	case canFlush && canHijack && canRead: // HTTP/1 requests.
		v := &http1Writer{statusRecorder: rec, flushCore: flush, Hijacker: hijack}
		return &v.statusRecorder, v

	case canFlush && canHijack && canPush:
		v := &flushHijackPushWriter{
			statusRecorder: rec,
			flushCore:      flush,
			Hijacker:       hijack,
			Pusher:         push,
		}
		return &v.statusRecorder, v

	case canFlush && canRead && canPush:
		v := &flushReadPushWriter{statusRecorder: rec, flushCore: flush, Pusher: push}
		return &v.statusRecorder, v

	case canFlush && canHijack:
		v := &flushHijackWriter{statusRecorder: rec, flushCore: flush, Hijacker: hijack}
		return &v.statusRecorder, v

	case canFlush && canRead:
		v := &flushReadWriter{statusRecorder: rec, flushCore: flush}
		return &v.statusRecorder, v

	case canFlush && canPush: // HTTP/2 requests.
		v := &flushPushWriter{statusRecorder: rec, flushCore: flush, Pusher: push}
		return &v.statusRecorder, v

	case canFlush:
		v := &flushWriter{statusRecorder: rec, flushCore: flush}
		return &v.statusRecorder, v

	case canHijack && canRead && canPush:
		v := &hijackReadPushWriter{statusRecorder: rec, Hijacker: hijack, Pusher: push}
		return &v.statusRecorder, v

	case canHijack && canRead:
		v := &hijackReadWriter{statusRecorder: rec, Hijacker: hijack}
		return &v.statusRecorder, v

	case canHijack && canPush:
		v := &hijackPushWriter{statusRecorder: rec, Hijacker: hijack, Pusher: push}
		return &v.statusRecorder, v

	case canRead && canPush:
		v := &readPushWriter{statusRecorder: rec, Pusher: push}
		return &v.statusRecorder, v

	case canHijack:
		v := &hijackWriter{statusRecorder: rec, Hijacker: hijack}
		return &v.statusRecorder, v

	case canRead:
		v := &readWriter{statusRecorder: rec}
		return &v.statusRecorder, v

	case canPush:
		v := &pushWriter{statusRecorder: rec, Pusher: push}
		return &v.statusRecorder, v

	default:
		v := &basicWriter{statusRecorder: rec}
		return &v.statusRecorder, v
	}
}
