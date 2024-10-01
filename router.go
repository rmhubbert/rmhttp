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
	Logger        Logger
	errorHandlers map[int]Handler
}

// NewRouter intialises, creates, and then returns a pointer to a Router.
func NewRouter(logger Logger) *Router {
	return &Router{
		Mux:           http.NewServeMux(),
		Logger:        logger,
		errorHandlers: make(map[int]Handler),
	}
}

// ServeHTTP allows the Router to fulfill the http.Handler interface, meaning that we can use it as
// a handler for the underlying HTTP request multiplexer (which by default is a http.ServeMux).
//
// Having the Router act as the primary handler allows us to inject our custom ResponseWriter and
// add the system logger to the Request (for use by any middleware).
//
// We can also intercept any error handlers returned by the underlying mux, and make sure that they
// are properly wrapped by the HTTP Error Handler and HTTP Logger (assuming the system is
// configured to enable them), as well as any middleware that was configured for the
// route.
//
// The Router is one of the few places where you will see ServeHTTP used instead of
// ServeHTTPWithError in the system.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handler returns the Handler that the Mux wants to use for this request.
	handler, pattern := rt.Mux.Handler(r)

	// If the pattern is empty, we probably have an internal error handler.
	if pattern == "" {
		if _, ok := handler.(Handler); !ok {
			// If we get here, then we have an http.Handler, which means that it is an internal error
			// handler. Check to see if we have a custom error handler for this error code, and use
			// that if so.
			cw := NewCaptureWriter(w)
			handler.ServeHTTP(cw, r)
			if h, ok := rt.errorHandlers[cw.Code]; ok {
				handler = h
			}
		}
	}

	handler.ServeHTTP(w, r)
}

// AddErrorHandler maps the passed response code and handler. These error handlers will be used
// instead of the http.Handler equivalents when available.
func (rt *Router) AddErrorHandler(code int, handler Handler) {
	rt.errorHandlers[code] = handler
}

// Handle registers the passed Route with the underlying HTTP request multiplexer.
func (rt *Router) Handle(method string, pattern string, handler Handler) {
	rt.Mux.Handle(fmt.Sprintf("%s %s", method, pattern), handler)
}
