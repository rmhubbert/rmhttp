package httplogger

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ------------------------------------------------------------------------------------------------
// FAKE RESPONSE WRITERS
//
// The middleware hands its own ResponseWriter wrapper to the next handler, so that wrapper must keep
// every optional interface (http.Flusher, http.Hijacker, io.ReaderFrom, http.Pusher) working, and it
// must never claim a capability the wrapped writer does not have.
//
// httptest.ResponseRecorder cannot be used to prove this: it advertises a capability set that
// matches neither a real HTTP/1 server writer nor a restricted one, so a "lying" wrapper still
// passes recorder-based tests. The fakes below each advertise exactly one capability combination,
// built up a single interface at a time, and count the calls they receive so tests can prove that a
// call made through the wrapper actually reached the underlying writer.
// ------------------------------------------------------------------------------------------------

// fakePlainWriter is a minimal http.ResponseWriter advertising no optional interface.
type fakePlainWriter struct {
	header      http.Header
	status      int
	headerCalls int
	body        bytes.Buffer
}

func newFakePlainWriter() *fakePlainWriter {
	return &fakePlainWriter{header: http.Header{}, status: http.StatusOK}
}

func (w *fakePlainWriter) Header() http.Header { return w.header }

func (w *fakePlainWriter) WriteHeader(status int) {
	w.headerCalls++
	w.status = status
}

func (w *fakePlainWriter) Write(p []byte) (int, error) { return w.body.Write(p) }

// fakeFlushWriter adds http.Flusher, like the bundled HTTP/2 server writer.
type fakeFlushWriter struct {
	fakePlainWriter
	flushes int
}

func newFakeFlushWriter() *fakeFlushWriter {
	return &fakeFlushWriter{fakePlainWriter: *newFakePlainWriter()}
}

func (w *fakeFlushWriter) Flush() { w.flushes++ }

// fakeHijackWriter adds http.Hijacker only, without http.Flusher.
type fakeHijackWriter struct {
	fakePlainWriter
	conn net.Conn
}

func newFakeHijackWriter(t *testing.T) *fakeHijackWriter {
	t.Helper()
	return &fakeHijackWriter{fakePlainWriter: *newFakePlainWriter(), conn: netPipeConn(t)}
}

func (w *fakeHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, newReadWriter(w.conn), nil
}

// fakeReadWriter adds io.ReaderFrom only (the sendfile fast path).
type fakeReadWriter struct {
	fakePlainWriter
	readFroms int64
}

func newFakeReadWriter() *fakeReadWriter {
	return &fakeReadWriter{fakePlainWriter: *newFakePlainWriter()}
}

func (w *fakeReadWriter) ReadFrom(src io.Reader) (int64, error) {
	n, err := io.Copy(&w.body, src)
	w.readFroms += n
	return n, err
}

// fakePushWriter adds http.Pusher only, matching the writer http.TimeoutHandler passes on.
type fakePushWriter struct {
	fakePlainWriter
	pushed []string
}

func newFakePushWriter() *fakePushWriter {
	return &fakePushWriter{fakePlainWriter: *newFakePlainWriter()}
}

func (w *fakePushWriter) Push(target string, _ *http.PushOptions) error {
	w.pushed = append(w.pushed, target)
	return nil
}

// fakeFlushHijackWriter adds http.Flusher and http.Hijacker, without io.ReaderFrom.
type fakeFlushHijackWriter struct {
	fakeFlushWriter
	conn net.Conn
}

func newFakeFlushHijackWriter(t *testing.T) *fakeFlushHijackWriter {
	t.Helper()
	return &fakeFlushHijackWriter{fakeFlushWriter: *newFakeFlushWriter(), conn: netPipeConn(t)}
}

func (w *fakeFlushHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.conn, newReadWriter(w.conn), nil
}

// fakeHTTP1Writer is the shape of a net/http HTTP/1 server writer: Flusher, Hijacker and
// io.ReaderFrom, but no http.Pusher.
type fakeHTTP1Writer struct {
	fakeFlushHijackWriter
	readFroms int64
}

func newFakeHTTP1Writer(t *testing.T) *fakeHTTP1Writer {
	t.Helper()
	return &fakeHTTP1Writer{fakeFlushHijackWriter: *newFakeFlushHijackWriter(t)}
}

func (w *fakeHTTP1Writer) ReadFrom(src io.Reader) (int64, error) {
	n, err := io.Copy(&w.body, src)
	w.readFroms += n
	return n, err
}

// fakeFullWriter advertises every optional interface at once.
type fakeFullWriter struct {
	fakeHTTP1Writer
	hijackErr error
	pushErr   error
	pushed    []string
}

func newFakeFullWriter(t *testing.T) *fakeFullWriter {
	t.Helper()
	return &fakeFullWriter{fakeHTTP1Writer: *newFakeHTTP1Writer(t)}
}

func (w *fakeFullWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.hijackErr != nil {
		return nil, nil, w.hijackErr
	}
	return w.conn, newReadWriter(w.conn), nil
}

func (w *fakeFullWriter) Push(target string, _ *http.PushOptions) error {
	if w.pushErr != nil {
		return w.pushErr
	}
	w.pushed = append(w.pushed, target)
	return nil
}

// netPipeConn returns a connection used as the hijacked connection under test, closed at cleanup.
func netPipeConn(t *testing.T) net.Conn {
	t.Helper()

	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	return server
}

func newReadWriter(conn net.Conn) *bufio.ReadWriter {
	return bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
}

// capabilities records which optional interfaces a ResponseWriter advertises.
type capabilities struct {
	flush    bool
	hijack   bool
	readFrom bool
	push     bool
}

// probeCapabilities reports the optional interfaces w advertises.
func probeCapabilities(w http.ResponseWriter) capabilities {
	_, flush := w.(http.Flusher)
	_, hijack := w.(http.Hijacker)
	_, readFrom := w.(io.ReaderFrom)
	_, push := w.(http.Pusher)

	return capabilities{flush: flush, hijack: hijack, readFrom: readFrom, push: push}
}

// Test_Middleware_ResponseWriterCapabilities pins the interface set the wrapper hands to the next
// handler: it must expose exactly the optional interfaces the underlying writer has. Advertising an
// interface the base lacks silently breaks streaming (a no-op Flush buffers SSE events forever) and
// panics websocket upgraders (a Hijack that returns a nil conn); dropping one breaks the handlers
// that legitimately use it.
func Test_Middleware_ResponseWriterCapabilities(t *testing.T) {
	tests := []struct {
		name    string
		base    func(t *testing.T) http.ResponseWriter
		current capabilities
	}{
		{
			name:    "plain writer exposes nothing",
			base:    func(_ *testing.T) http.ResponseWriter { return newFakePlainWriter() },
			current: capabilities{},
		},
		{
			name:    "flusher only",
			base:    func(_ *testing.T) http.ResponseWriter { return newFakeFlushWriter() },
			current: capabilities{flush: true},
		},
		{
			name:    "hijacker only",
			base:    func(t *testing.T) http.ResponseWriter { return newFakeHijackWriter(t) },
			current: capabilities{hijack: true},
		},
		{
			name:    "reader only",
			base:    func(_ *testing.T) http.ResponseWriter { return newFakeReadWriter() },
			current: capabilities{readFrom: true},
		},
		{
			name:    "pusher only like http.TimeoutHandler",
			base:    func(_ *testing.T) http.ResponseWriter { return newFakePushWriter() },
			current: capabilities{push: true},
		},
		{
			name:    "flusher and hijacker",
			base:    func(t *testing.T) http.ResponseWriter { return newFakeFlushHijackWriter(t) },
			current: capabilities{flush: true, hijack: true},
		},
		{
			name:    "net/http HTTP/1 server writer shape",
			base:    func(t *testing.T) http.ResponseWriter { return newFakeHTTP1Writer(t) },
			current: capabilities{flush: true, hijack: true, readFrom: true},
		},
		{
			name:    "all optional interfaces",
			base:    func(t *testing.T) http.ResponseWriter { return newFakeFullWriter(t) },
			current: capabilities{flush: true, hijack: true, readFrom: true, push: true},
		},
		{
			name:    "flusher and reader",
			base:    func(_ *testing.T) http.ResponseWriter { return newFakeFlushReadWriter() },
			current: capabilities{flush: true, readFrom: true},
		},
		{
			name:    "flusher and pusher",
			base:    func(_ *testing.T) http.ResponseWriter { return newFakeFlushPushWriter() },
			current: capabilities{flush: true, push: true},
		},
		{
			name:    "flusher, reader and pusher",
			base:    func(_ *testing.T) http.ResponseWriter { return newFakeFlushReadPushWriter() },
			current: capabilities{flush: true, readFrom: true, push: true},
		},
		{
			name:    "hijacker and reader",
			base:    func(t *testing.T) http.ResponseWriter { return newFakeHijackReadWriter(t) },
			current: capabilities{hijack: true, readFrom: true},
		},
		{
			name:    "hijacker and pusher",
			base:    func(t *testing.T) http.ResponseWriter { return newFakeHijackPushWriter(t) },
			current: capabilities{hijack: true, push: true},
		},
		{
			name:    "reader and pusher",
			base:    func(_ *testing.T) http.ResponseWriter { return newFakeReadPushWriter() },
			current: capabilities{readFrom: true, push: true},
		},
		{
			name:    "hijacker, reader and pusher",
			base:    func(t *testing.T) http.ResponseWriter { return newFakeHijackReadPushWriter(t) },
			current: capabilities{hijack: true, readFrom: true, push: true},
		},
		{
			name:    "flusher, hijacker and pusher",
			base:    func(t *testing.T) http.ResponseWriter { return newFakeFlushHijackPushWriter(t) },
			current: capabilities{flush: true, hijack: true, push: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base := test.base(t)
			want := probeCapabilities(base)
			assert.Equal(
				t,
				want,
				test.current,
				"test fixture drifted from its documented capability set",
			)

			var got capabilities
			handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				got = probeCapabilities(w)
				_, _ = w.Write([]byte("body"))
			}))
			handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

			assert.Equal(t, want, got,
				"the wrapper must advertise exactly the underlying writer's optional interfaces")
		})
	}
}

// Test_Middleware_WriterDelegation proves that every capability call made against the writer handed
// to the handler reaches the underlying writer, either through the wrapper method or through
// http.ResponseController (which relies on Unwrap).
func Test_Middleware_WriterDelegation(t *testing.T) {
	t.Run("flush reaches the base writer", func(t *testing.T) {
		base := newFakeFlushWriter()

		var controllerErr error
		handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.(http.Flusher).Flush()
			controllerErr = http.NewResponseController(w).Flush()
		}))
		handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

		require.NoError(t, controllerErr, "ResponseController.Flush must work through the wrapper")
		assert.Equal(t, 2, base.flushes, "both flush paths must reach the underlying writer")
	})

	t.Run("hijack reaches the base writer", func(t *testing.T) {
		base := newFakeFlushHijackWriter(t)
		var directConn, controllerConn net.Conn
		handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			directConn, _, _ = w.(http.Hijacker).Hijack()
			controllerConn, _, _ = http.NewResponseController(w).Hijack()
		}))
		handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

		assert.Same(t, base.conn, directConn, "Hijack must hand back the base connection")
		assert.Same(
			t,
			base.conn,
			controllerConn,
			"ResponseController must unwrap to the base connection",
		)
	})

	t.Run("hijack errors are propagated honestly", func(t *testing.T) {
		base := newFakeFullWriter(t)
		base.hijackErr = http.ErrNotSupported

		var gotErr error
		handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _, gotErr = w.(http.Hijacker).Hijack()
		}))
		handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

		assert.ErrorIs(t, gotErr, http.ErrNotSupported)
	})

	t.Run("read from is delegated and counted", func(t *testing.T) {
		base := newFakeReadWriter()
		payload := "sendfile-body"
		handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.(io.ReaderFrom).ReadFrom(bytes.NewReader([]byte(payload)))
			assert.NoError(t, err)
		}))

		out.Reset()
		handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

		assert.Equal(
			t,
			payload,
			base.body.String(),
			"ReadFrom must stream through to the base writer",
		)
		assert.Contains(t, out.String(), `"size":`+strconv.Itoa(len(payload)),
			"bytes copied by ReadFrom must be counted in the logged size")
	})

	t.Run("push reaches the base writer", func(t *testing.T) {
		base := newFakePushWriter()
		handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			assert.NoError(t, w.(http.Pusher).Push("/style.css", nil))
		}))
		handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

		assert.Equal(t, []string{"/style.css"}, base.pushed)
	})

	t.Run("headers are shared with the base writer", func(t *testing.T) {
		base := newFakePlainWriter()
		handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Test", "value")
			_, _ = w.Write([]byte("body"))
		}))
		handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

		assert.Equal(t, "value", base.header.Get("X-Test"))
	})

	t.Run("unwrap returns the base writer", func(t *testing.T) {
		base := newFakeFlushWriter()

		var unwrapped http.ResponseWriter
		handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if u, ok := w.(interface{ Unwrap() http.ResponseWriter }); ok {
				unwrapped = u.Unwrap()
			} else {
				t.Error("the wrapper must implement Unwrap so http.ResponseController works")
			}
		}))
		handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

		assert.Same(t, http.ResponseWriter(base), unwrapped)
	})
}

// decodeLogged writes one request through the middleware and decodes the single JSON entry it must
// emit. Sequential use only: the shared package log buffer is not concurrency-safe.
func decodeLogged(t *testing.T, base http.ResponseWriter, handler http.Handler) logEntry {
	t.Helper()

	out.Reset()
	Middleware()(handler).ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

	entry := logEntry{}
	require.NoError(
		t,
		json.Unmarshal(out.Bytes(), &entry),
		"one JSON log entry expected, got %q",
		out.String(),
	)

	return entry
}

// Test_Middleware_StatusCapture pins the status and size the middleware reports, matching the
// semantics of the capture implementation it replaces: the first non-informational WriteHeader wins,
// a bare Write commits an implicit 200, and size is the total number of body bytes written.
func Test_Middleware_StatusCapture(t *testing.T) {
	tests := []struct {
		name          string
		handler       http.HandlerFunc
		wantStatus    int
		wantSizeBytes int
	}{
		{
			name: "implicit 200 on write without WriteHeader",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("body"))
			},
			wantStatus:    http.StatusOK,
			wantSizeBytes: len("body"),
		},
		{
			name: "explicit status wins",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTeapot)
				_, _ = w.Write([]byte("body"))
			},
			wantStatus:    http.StatusTeapot,
			wantSizeBytes: len("body"),
		},
		{
			name: "informational response does not commit the status",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusEarlyHints)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("body"))
			},
			wantStatus:    http.StatusOK,
			wantSizeBytes: len("body"),
		},
		{
			name: "the first non-informational status wins",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("nope"))
			},
			wantStatus:    http.StatusNotFound,
			wantSizeBytes: len("nope"),
		},
		{
			name: "a status set after the body does not overwrite the implicit 200",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("body"))
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantStatus:    http.StatusOK,
			wantSizeBytes: len("body"),
		},
		{
			name: "an empty body is logged as zero bytes",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantStatus:    http.StatusNoContent,
			wantSizeBytes: 0,
		},
		{
			name: "repeated writes are summed",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("aaaa"))
				_, _ = w.Write([]byte("bb"))
			},
			wantStatus:    http.StatusOK,
			wantSizeBytes: 6,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			entry := decodeLogged(t, newFakePlainWriter(), test.handler)

			assert.Equal(t, test.wantStatus, entry.Status)
			assert.Equal(t, test.wantSizeBytes, entry.Size)
		})
	}
}

// Test_Middleware_WritesPassThroughToBaseWriter proves every WriteHeader/Write call made by the
// handler still reaches the underlying writer exactly once, which is what keeps the response correct
// for the client and keeps superfluous-WriteHeader detection intact in the server.
func Test_Middleware_WritesPassThroughToBaseWriter(t *testing.T) {
	base := newFakeFlushWriter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("one"))
		_, _ = w.Write([]byte("two"))
	})

	out.Reset()
	Middleware()(handler).ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

	assert.Equal(t, 1, base.headerCalls, "WriteHeader must be forwarded exactly once")
	assert.Equal(t, http.StatusCreated, base.status)
	assert.Equal(t, "onetwo", base.body.String(), "every Write must be forwarded in order")
	assert.Contains(t, out.String(), `"size":6`)
}

// Test_Middleware_SSEFlushScenario pins the streaming behaviour that a lying wrapper destroys: an
// SSE-style handler flushes between events, using both the direct http.Flusher type assertion that
// typical SSE code uses and http.NewResponseController, which relies on Unwrap. If either path is
// severed, events sit in a buffer and never reach clients while every test still passes.
func Test_Middleware_SSEFlushScenario(t *testing.T) {
	base := newFakeFlushWriter()

	events := []string{"data: first\n\n", "data: second\n\n"}

	var (
		flusherAdvertised bool
		controllerErr     error
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			flusherAdvertised = false
			return
		}

		flusherAdvertised = true
		controller := http.NewResponseController(w)

		_, _ = io.WriteString(w, events[0])
		flusher.Flush()

		controllerErr = controller.Flush()

		_, _ = io.WriteString(w, events[1])
		flusher.Flush()
	})

	entry := decodeLogged(t, base, handler)

	require.True(
		t,
		flusherAdvertised,
		"an SSE handler must be able to assert http.Flusher on the writer it receives",
	)
	require.NoError(t, controllerErr, "ResponseController.Flush must work through the wrapper")
	assert.Equal(t, 3, base.flushes, "all three flush calls must reach the underlying writer")
	assert.Equal(
		t,
		strings.Join(events, ""),
		base.body.String(),
		"events must be delivered in order",
	)
	assert.Equal(t, http.StatusOK, entry.Status)
	assert.Equal(t, len(events[0])+len(events[1]), entry.Size)
}

// Test_Middleware_HonestInterfaceSet is the negative half of the capability contract: when the
// underlying writer implements no optional interface, the writer handed to the handler must fail
// every optional type assertion. A single wrapper struct with unconditional Flush/Hijack/Push
// methods would pass every normal green test here while breaking SSE (silent no-op flush) and
// websockets (nil-conn panic) in production.
func Test_Middleware_HonestInterfaceSet(t *testing.T) {
	base := newFakePlainWriter()

	var (
		handlerRan     bool
		writerCaps     capabilities
		controllerErrs []string
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerRan = true
		writerCaps = probeCapabilities(w)

		// http.ResponseController must report "not supported" rather than pretend to work.
		rc := http.NewResponseController(w)
		if err := rc.Flush(); err != nil {
			controllerErrs = append(controllerErrs, "flush")
		}
		if _, _, err := rc.Hijack(); err != nil {
			controllerErrs = append(controllerErrs, "hijack")
		}
	})

	entry := decodeLogged(t, base, handler)

	assert.True(t, handlerRan, "the handler must run with an unrestricted body writer")
	assert.Equal(
		t,
		capabilities{},
		writerCaps,
		"the wrapper must not advertise capabilities the base lacks",
	)
	assert.Equal(t, []string{"flush", "hijack"}, controllerErrs,
		"ResponseController must report unsupported operations on a plain writer")
	assert.Equal(t, http.StatusOK, entry.Status)
	assert.Empty(t, base.body.String(), "nothing was written by the handler")
}

// pushCounter is embedded by fakes that advertise http.Pusher.
type pushCounter struct{ pushed []string }

func (p *pushCounter) Push(target string, _ *http.PushOptions) error {
	p.pushed = append(p.pushed, target)
	return nil
}

// fakeFlushReadWriter advertises Flusher and io.ReaderFrom.
type fakeFlushReadWriter struct {
	fakeFlushWriter
	readFroms int64
}

func newFakeFlushReadWriter() *fakeFlushReadWriter {
	return &fakeFlushReadWriter{fakeFlushWriter: *newFakeFlushWriter()}
}

func (w *fakeFlushReadWriter) ReadFrom(src io.Reader) (int64, error) {
	n, err := io.Copy(&w.body, src)
	w.readFroms += n
	return n, err
}

// fakeFlushPushWriter advertises Flusher and Pusher.
type fakeFlushPushWriter struct {
	fakeFlushWriter
	pushCounter
}

func newFakeFlushPushWriter() *fakeFlushPushWriter {
	return &fakeFlushPushWriter{fakeFlushWriter: *newFakeFlushWriter()}
}

// fakeFlushReadPushWriter advertises Flusher, io.ReaderFrom and Pusher.
type fakeFlushReadPushWriter struct {
	fakeFlushReadWriter
	pushCounter
}

func newFakeFlushReadPushWriter() *fakeFlushReadPushWriter {
	return &fakeFlushReadPushWriter{fakeFlushReadWriter: *newFakeFlushReadWriter()}
}

// fakeFlushHijackPushWriter advertises Flusher, Hijacker and Pusher, but not io.ReaderFrom. This is
// the shape of the CaptureWriter in the module's root package.
type fakeFlushHijackPushWriter struct {
	fakeFlushHijackWriter
	pushCounter
}

func newFakeFlushHijackPushWriter(t *testing.T) *fakeFlushHijackPushWriter {
	t.Helper()
	return &fakeFlushHijackPushWriter{fakeFlushHijackWriter: *newFakeFlushHijackWriter(t)}
}

// fakeHijackReadWriter advertises Hijacker and io.ReaderFrom, but cannot be flushed.
type fakeHijackReadWriter struct {
	fakeHijackWriter
	readFroms int64
}

func newFakeHijackReadWriter(t *testing.T) *fakeHijackReadWriter {
	t.Helper()
	return &fakeHijackReadWriter{fakeHijackWriter: *newFakeHijackWriter(t)}
}

func (w *fakeHijackReadWriter) ReadFrom(src io.Reader) (int64, error) {
	n, err := io.Copy(&w.body, src)
	w.readFroms += n
	return n, err
}

// fakeHijackPushWriter advertises Hijacker and Pusher, without Flusher.
type fakeHijackPushWriter struct {
	fakeHijackWriter
	pushCounter
}

func newFakeHijackPushWriter(t *testing.T) *fakeHijackPushWriter {
	t.Helper()
	return &fakeHijackPushWriter{fakeHijackWriter: *newFakeHijackWriter(t)}
}

// fakeReadPushWriter advertises io.ReaderFrom and Pusher, without Flusher.
type fakeReadPushWriter struct {
	fakeReadWriter
	pushCounter
}

func newFakeReadPushWriter() *fakeReadPushWriter {
	return &fakeReadPushWriter{fakeReadWriter: *newFakeReadWriter()}
}

// fakeHijackReadPushWriter advertises Hijacker, io.ReaderFrom and Pusher, without Flusher.
type fakeHijackReadPushWriter struct {
	fakeHijackReadWriter
	pushCounter
}

func newFakeHijackReadPushWriter(t *testing.T) *fakeHijackReadPushWriter {
	t.Helper()
	return &fakeHijackReadPushWriter{fakeHijackReadWriter: *newFakeHijackReadWriter(t)}
}

// Test_Middleware_UnwrapReachesBaseWriter proves the escape hatch that makes under-advertising safe:
// unwrapping the writer handed to the handler returns the original writer with all of its
// capabilities.
func Test_Middleware_UnwrapReachesBaseWriter(t *testing.T) {
	base := newFakeHijackPushWriter(t)

	var (
		unwrapped   http.ResponseWriter
		sameCaps    capabilities
		unwrappedOK bool
	)

	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		u, ok := w.(interface{ Unwrap() http.ResponseWriter })
		unwrappedOK = ok
		if ok {
			unwrapped = u.Unwrap()
			sameCaps = probeCapabilities(unwrapped)
		}
	}))
	handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

	require.True(t, unwrappedOK, "the wrapper must implement Unwrap")
	assert.Same(t, http.ResponseWriter(base), unwrapped)
	assert.Equal(t, capabilities{hijack: true, push: true}, sameCaps,
		"the base writer keeps every capability through Unwrap")
}

// fakeExtendedWriter is the shape of a modern net/http server writer: besides Flusher it implements
// the http.ResponseController capabilities FlushError, SetReadDeadline, SetWriteDeadline and
// EnableFullDuplex.
type fakeExtendedWriter struct {
	fakeFlushWriter
	flushErr    error
	deadlines   []time.Time
	fullDuplex  int
	reportedErr error
}

func newFakeExtendedWriter() *fakeExtendedWriter {
	return &fakeExtendedWriter{fakeFlushWriter: *newFakeFlushWriter()}
}

func (w *fakeExtendedWriter) FlushError() error {
	w.flushes++
	w.reportedErr = w.flushErr

	return w.flushErr
}

func (w *fakeExtendedWriter) SetReadDeadline(t time.Time) error {
	w.deadlines = append(w.deadlines, t)
	return nil
}

func (w *fakeExtendedWriter) SetWriteDeadline(t time.Time) error {
	w.deadlines = append(w.deadlines, t)
	return nil
}

func (w *fakeExtendedWriter) EnableFullDuplex() error {
	w.fullDuplex++
	return nil
}

// Test_Middleware_ResponseControllerCapabilities walks the wrapper chain: the deadline and
// full-duplex capabilities are deliberately not mirrored by the wrapper variants, so
// http.ResponseController must reach them through Unwrap.
func Test_Middleware_ResponseControllerCapabilities(t *testing.T) {
	base := newFakeExtendedWriter()
	deadline := time.Now().Add(time.Minute)

	var (
		writeDeadlineErr error
		fullDuplexErr    error
	)

	handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rc := http.NewResponseController(w)
		writeDeadlineErr = rc.SetWriteDeadline(deadline)
		fullDuplexErr = rc.EnableFullDuplex()
	}))
	handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

	require.NoError(
		t,
		writeDeadlineErr,
		"SetWriteDeadline must reach the base writer through Unwrap",
	)
	require.NoError(t, fullDuplexErr, "EnableFullDuplex must reach the base writer through Unwrap")
	assert.Equal(t, []time.Time{deadline}, base.deadlines)
	assert.Equal(t, 1, base.fullDuplex)
}

// Test_Middleware_FlushErrorFidelity pins that flushing through the wrapper is as informative as
// flushing the writer directly. http.ResponseController.Flush prefers an
// interface{ FlushError() error } implementation over a plain http.Flusher, so a writer that reports
// flush errors must keep reporting them through the wrapper, and a writer that cannot must still
// flush and report nil.
func Test_Middleware_FlushErrorFidelity(t *testing.T) {
	t.Run("base reports flush errors", func(t *testing.T) {
		base := newFakeExtendedWriter()
		base.flushErr = io.ErrClosedPipe

		var gotErr error
		handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			gotErr = http.NewResponseController(w).Flush()
		}))
		handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

		require.ErrorIs(
			t,
			gotErr,
			io.ErrClosedPipe,
			"the base writer's flush error must be surfaced",
		)
		assert.Equal(t, 1, base.flushes)
	})

	t.Run("base cannot report flush errors", func(t *testing.T) {
		base := newFakeFlushWriter()

		var (
			gotErr error
			probed bool
		)

		handler := Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, probed = w.(interface{ FlushError() error })
			gotErr = http.NewResponseController(w).Flush()
		}))
		handler.ServeHTTP(base, newBenchRequest("203.0.113.7:52000", nil, ""))

		assert.True(t, probed, "flush-capable variants answer the FlushError probe")
		require.NoError(t, gotErr, "nil is the same answer an unwrapped bare Flusher produces")
		assert.Equal(t, 1, base.flushes, "the probe must still flush")
	})
}
