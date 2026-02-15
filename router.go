package rmhttp

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
)

// ------------------------------------------------------------------------------------------------
// ROUTER
// ------------------------------------------------------------------------------------------------

// The Router loads Routes into the underlying HTTP request multiplexer, as well as handling each
// request, ensuring that ResponseWriter and Request objects are properly configured. The Router
// also manages custom error handlers to ensure that the HTTP Error Handler can operate
// properly.
type Router struct {
	Mux               *http.ServeMux
	errorHandlers     sync.Map
	errorHandlerCount int32
}

// NewRouter intialises, creates, and then returns a pointer to a Router.
func NewRouter() *Router {
	return &Router{
		Mux:           http.NewServeMux(),
		errorHandlers: sync.Map{},
	}
}

// ServeHTTP allows the Router to fulfill the http.Handler interface, meaning that we can use it as
// a handler for the underlying HTTP request multiplexer (which by default is a http.ServeMux).
//
// We also intercept any error handlers returned by the underlying mux, and replace them with any
// custom error handlers that have been registered.
//
// Note: CaptureWriter instances are created per-request and should not be shared across concurrent
// requests. The sync.Map provides thread-safe access to the errorHandlers map, and errorCount
// provides a thread-safe count of error handlers without needing a mutex.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// If there are no error handlers, we can just use the underlying mux.
	if atomic.LoadInt32(&rt.errorHandlerCount) == 0 {
		rt.Mux.ServeHTTP(w, r)
		return
	}

	handler, pattern := rt.Mux.Handler(r)

	// When ServeMux.Handler() returns an empty pattern, it means either:
	// 1. No route matched (404)
	// 2. Method not allowed for matched pattern (405)
	// In both cases, we capture the response and check for custom error handlers
	if pattern == "" && handler != nil {
		// If pattern is empty, we have an internal error handler (404 or 405).
		// Check to see if we have a custom error handler for this error code.
		cw := NewCaptureWriter(w)
		cw.PassThrough = false
		handler.ServeHTTP(cw, r)

		// Only use error handler if we captured a non-200 status
		if cw.Code != 0 && cw.Code != http.StatusOK {
			if h, ok := rt.errorHandlers.Load(cw.Code); ok {
				handler = h.(http.Handler)
			}
		}

		// Use the custom error handler
		handler.ServeHTTP(w, r)
		return
	}

	// For normal requests, use the mux's ServeHTTP to ensure path values are extracted
	rt.Mux.ServeHTTP(w, r)
}

// AddErrorHandler maps the passed response code and handler. These error handlers will be used
// instead of the http.Handler equivalents when available.
func (rt *Router) AddErrorHandler(code int, handler http.Handler) {
	_, loaded := rt.errorHandlers.LoadOrStore(code, handler)
	if !loaded {
		atomic.AddInt32(&rt.errorHandlerCount, 1)
	}
}

// Handle registers the passed Route with the underlying HTTP request multiplexer.
func (rt *Router) Handle(method string, pattern string, handler http.Handler) {
	rt.Mux.Handle(fmt.Sprintf("%s %s", method, pattern), handler)
}
