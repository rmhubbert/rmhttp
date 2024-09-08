package rmhttp

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"
)

// ------------------------------------------------------------------------------------------------
// TIMEOUT
// ------------------------------------------------------------------------------------------------

// Timeout encapsulates a duration and message that should be used for applying timeouts to
// Route handlers, with a specific error message.
type Timeout struct {
	Duration time.Duration
	Message  string
	Enabled  bool
}

// NewTimeout creates, initialises and returns a pointer to a Timeout.
func NewTimeout(duration time.Duration, message string) Timeout {
	return Timeout{
		Duration: duration,
		Message:  message,
		Enabled:  true,
	}
}

// ------------------------------------------------------------------------------------------------
// TIMEOUT MIDDLEWARE
// ------------------------------------------------------------------------------------------------

// TimeoutMiddleware creates, initialises and returns a middleware function that will wrap the next
// handler in the stack with a timeout handler.
func TimeoutMiddleware(timeout Timeout) MiddlewareFunc {
	return MiddlewareFunc(func(next Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			th := NewTimeoutHandler(next, timeout)
			return th.ServeHTTPWithError(w, r)
		})
	})
}

// ------------------------------------------------------------------------------------------------
// TIMEOUT HANDLER
// ------------------------------------------------------------------------------------------------

// A TimeoutHandler implements the Handler interface. Its primary purpose is to wrap an HTTPHandler
// and provide an execution timeout.
//
// Every route handler, with the exception of those dynamically generated in response to an
// internal error, will be wrapped with a TimeoutHandler. There is no configurable option
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
type TimeoutHandler struct {
	timeout Timeout
	handler Handler
}

// TimeoutHandler creates, initialises and returns a pointer to a new timeoutHandler.
func NewTimeoutHandler(
	handler Handler,
	timeout Timeout,
) *TimeoutHandler {
	return &TimeoutHandler{
		handler: handler,
		timeout: timeout,
	}
}

// ServeHTTP fulfills the http.Handler interface but is rarely used. You should prefer
// ServeHTTPWithError wherever possible.
func (h *TimeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = h.ServeHTTPWithError(w, r)
}

// ServeHTTPWithError implements the rmhttp.Handler interface and handles the actual timeout
// management.
//
// This function is a simplified version of the net/http version, with the addition of error
// returning.
func (h *TimeoutHandler) ServeHTTPWithError(w http.ResponseWriter, r *http.Request) error {
	ctx, cancelCtx := context.WithTimeout(r.Context(), h.timeout.Duration)
	defer cancelCtx()
	r = r.WithContext(ctx)

	done := make(chan error)
	panicChan := make(chan any, 1)
	cw := NewCaptureWriter(w)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
				close(panicChan)
			}
		}()
		done <- h.handler.ServeHTTPWithError(cw, r)
		close(done)
	}()

	select {
	case p := <-panicChan:
		panic(p)
	case e := <-done:
		cw.Mu.Lock()
		defer cw.Mu.Unlock()
		cw.Persist()
		return e
	case <-ctx.Done():
		cw.Mu.Lock()
		defer cw.Mu.Unlock()

		switch err := ctx.Err(); err {
		case context.DeadlineExceeded:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, h.timeout.Message)
			return NewHTTPError(errors.New(h.timeout.Message), http.StatusServiceUnavailable)
		default:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, err.Error())
			return NewHTTPError(err, http.StatusServiceUnavailable)
		}
	}
}
