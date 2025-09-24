package rmhttp

import (
	"fmt"
	"net/http"
)

// ------------------------------------------------------------------------------------------------
// ROUTER
// ------------------------------------------------------------------------------------------------

// The Router loads Routes into the underlying HTTP request multiplexer, as well as handling each
// request, ensuring that ResponseWriter and Request objects are properly configured. The Router
// also manages custom error handlers to ensure that the HTTP Error Handler can operate
// properly.
type Router struct {
	Mux           *http.ServeMux
	errorHandlers map[int]http.Handler
}

// NewRouter intialises, creates, and then returns a pointer to a Router.
func NewRouter() *Router {
	return &Router{
		Mux:           http.NewServeMux(),
		errorHandlers: make(map[int]http.Handler),
	}
}

// ServeHTTP allows the Router to fulfill the http.Handler interface, meaning that we can use it as
// a handler for the underlying HTTP request multiplexer (which by default is a http.ServeMux).
//
// We also intercept any error handlers returned by the underlying mux, and replace them with any
// custom error handlers that have been registered.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(rt.errorHandlers) == 0 {
		rt.Mux.ServeHTTP(w, r)
		return
	}

	handler, pattern := rt.Mux.Handler(r)
	if pattern == "" && handler != nil {
		// If pattern is empty, we have an internal error handler. Check to see if we have a custom
		// error handler for this error code, and use that if we do.
		cw := NewCaptureWriter(w)
		cw.PassThrough = false
		handler.ServeHTTP(cw, r)
		if h, ok := rt.errorHandlers[cw.Code]; ok {
			handler = h
		}
	}
	handler.ServeHTTP(w, r)
}

// AddErrorHandler maps the passed response code and handler. These error handlers will be used
// instead of the http.Handler equivalents when available.
func (rt *Router) AddErrorHandler(code int, handler http.Handler) {
	rt.errorHandlers[code] = handler
}

// Handle registers the passed Route with the underlying HTTP request multiplexer.
func (rt *Router) Handle(method string, pattern string, handler http.Handler) {
	rt.Mux.Handle(fmt.Sprintf("%s %s", method, pattern), handler)
}
