package rmhttp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

// ------------------------------------------------------------------------------------------------
// TIMEOUT
// ------------------------------------------------------------------------------------------------

// A Timeout encapsulates a duration and message that should be used for applying timeouts to
// Route handlers, with a specific error message.
type Timeout struct {
	duration time.Duration
	message  string
}

// newTimeout creates, initialises and returns a pointer to a Timeout.
func NewTimeout(duration time.Duration, message string) Timeout {
	return Timeout{
		duration: duration,
		message:  message,
	}
}

// ------------------------------------------------------------------------------------------------
// TIMEOUT SERVICE
// ------------------------------------------------------------------------------------------------

// A timeoutService supplies functionality for applying timeouts to route handlers and ensuring that
// the Server TCP timeout is at least as long as the longest route timeout.
type timeoutService struct {
	config TimeoutConfig
	logger Logger
}

// newTimeoutService creates, initialises and returns a pointer to a new timeoutService.
func newTimeoutService(config TimeoutConfig, logger Logger) *timeoutService {
	return &timeoutService{
		config: config,
		logger: logger,
	}
}

// applyTimeout wraps the passed handler with a timeoutHandler initialised with the passed
// duration and message.
func (tos *timeoutService) applyTimeout(handler Handler, timeout Timeout) Handler {
	if timeout.duration <= 0 {
		return handler
	}
	// TODO: update Server TCP timeouts, if neceassary.
	return TimeoutHandler(handler, timeout, tos.logger)
}

// ------------------------------------------------------------------------------------------------
// TIMEOUT HANDLER
// ------------------------------------------------------------------------------------------------

// A timeoutHandler implements the Handler interface. Its primary purpose is to wrap an HTTPHandler
// and provide an execution timeout.
//
// Every route handler, with the exception of those dynamically generated in response to an
// internal error, will be wrapped with a timeoutHandler. There is no configurable option
// to turn this off for security reasons, but a user could set it to a very large
// duration, if desired.
//
// This implementation feels very hacky but I can't think of a better way to implement per route
// timeouts with our custom HandlerFunc error returning. This is basically just a simplified
// version of the net/http implementation with some minor changes to accomodate passing
// errors through the timeout handler. It's necessary as net/http sets this
// functionality to be unexportable, so we can't just embed timeout
// handlers into our own structs.
//
// https://cs.opensource.google/go/go/+/master:src/net/http/server.go
type timeoutHandler struct {
	timeout Timeout
	handler Handler
	logger  Logger
}

// TimeoutHandler creates, initialises and returns a pointer to a new timeoutHandler.
func TimeoutHandler(
	handler Handler,
	timeout Timeout,
	logger Logger,
) *timeoutHandler {
	return &timeoutHandler{
		handler: handler,
		timeout: timeout,
		logger:  logger,
	}
}

// ServeHTTP fulfills the http.Handler interface but is rarely used. You should prefer
// ServeHTTPWithError wherever possible.
func (h *timeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = h.ServeHTTPWithError(w, r)
}

// ServeHTTPWithError implements the rmhttp.Handler interface and handles the actual timeout
// management.
//
// This function is a simplified version of the net/http version, with the addition of error
// returning, and the removal of direct ResponseWriter calls (this should happen in the
// http error handler instead).
func (h *timeoutHandler) ServeHTTPWithError(w http.ResponseWriter, r *http.Request) error {
	ctx, cancelCtx := context.WithTimeout(r.Context(), h.timeout.duration)
	defer cancelCtx()
	r = r.WithContext(ctx)

	done := make(chan error)
	panicChan := make(chan any, 1)
	tw := &timeoutWriter{
		w:      w,
		h:      make(http.Header),
		req:    r,
		logger: h.logger,
	}

	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
				close(panicChan)
			}
		}()
		done <- h.handler.ServeHTTPWithError(tw, r)
		close(done)
	}()

	select {
	case p := <-panicChan:
		panic(p)
	case e := <-done:
		tw.mu.Lock()
		defer tw.mu.Unlock()
		dst := w.Header()
		for k, vv := range tw.h {
			dst[k] = vv
		}
		if !tw.wroteHeader {
			tw.code = http.StatusOK
		}
		w.WriteHeader(tw.code)
		_, _ = w.Write(tw.wbuf.Bytes())
		return e
	case <-ctx.Done():
		switch err := ctx.Err(); err {
		case context.DeadlineExceeded:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, h.timeout.message)
			return NewHTTPError(h.timeout.message, http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, err.Error())
			return NewHTTPError(err.Error(), http.StatusServiceUnavailable)
		}
	}
}

// A timeoutWriter is used in the timeoutHandler instead of the http.ResponseWriter to capture
// header abd body writes from the passed handler, so that we can then set them manually,
// depending on whether or not the request times out.
type timeoutWriter struct {
	w           http.ResponseWriter
	h           http.Header
	wbuf        bytes.Buffer
	req         *http.Request
	logger      Logger
	mu          sync.Mutex
	err         error
	wroteHeader bool
	code        int
}

var _ http.Pusher = (*timeoutWriter)(nil)

// Push implements the [Pusher] interface.
func (tw *timeoutWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := tw.w.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

// Header implements part of the http.ResponseWriter interface.
func (tw *timeoutWriter) Header() http.Header { return tw.h }

// Write implements part of the http.ResponseWriter interface.
func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.err != nil {
		return 0, tw.err
	}
	if !tw.wroteHeader {
		tw.writeHeaderLocked(http.StatusOK)
	}
	return tw.wbuf.Write(p)
}

// writeHeaderLocked checks if the status code has already been written. If not, it will write the
// passed code to the response.
func (tw *timeoutWriter) writeHeaderLocked(code int) {
	checkWriteHeaderCode(code)

	switch {
	case tw.err != nil:
		return
	case tw.wroteHeader:
		if tw.req != nil {
			caller := relevantCaller()
			tw.logger.Error(
				fmt.Sprintf("http: superfluous response.WriteHeader call from %s (%s:%d)",
					caller.Function,
					path.Base(caller.File),
					caller.Line),
			)
		}
	default:
		tw.wroteHeader = true
		tw.code = code
	}
}

// WriteHeader implements part of the http.ResponseWriter interface.
func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.writeHeaderLocked(code)
}

// checkWriteHeaderCode makes sure that the passed status code is within a valid range for HTTP
// status codes.
func checkWriteHeaderCode(code int) {
	// Issue 22880: require valid WriteHeader status codes.
	// For now we only enforce that it's three digits.
	// In the future we might block things over 599 (600 and above aren't defined
	// at https://httpwg.org/specs/rfc7231.html#status.codes).
	// But for now any three digits.
	//
	// We used to send "HTTP/1.1 000 0" on the wire in responses but there's
	// no equivalent bogus thing we can realistically send in HTTP/2,
	// so we'll consistently panic instead and help people find their bugs
	// early. (We can't return an error from WriteHeader even if we wanted to.)
	if code < 100 || code > 999 {
		panic(fmt.Sprintf("invalid WriteHeader code %v", code))
	}
}

// relevantCaller searches the call stack for the first function outside of net/http.
// The purpose of this function is to provide more helpful error messages.
func relevantCaller() runtime.Frame {
	pc := make([]uintptr, 16)
	n := runtime.Callers(1, pc)
	frames := runtime.CallersFrames(pc[:n])
	var frame runtime.Frame
	for {
		frame, more := frames.Next()
		if !strings.HasPrefix(frame.Function, "net/http.") {
			return frame
		}
		if !more {
			break
		}
	}
	return frame
}
