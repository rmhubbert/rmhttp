package rmhttp

import "net/http"

// ------------------------------------------------------------------------------------------------
// HANDLER INTERFACE
// ------------------------------------------------------------------------------------------------
// Handler extends the http.Handler interface with ServeHTTPWithError, allowing Handlers to return
// errors.
type Handler interface {
	http.Handler
	ServeHTTPWithError(http.ResponseWriter, *http.Request) error
}

// ------------------------------------------------------------------------------------------------
// HANDLERFUNC
// ------------------------------------------------------------------------------------------------
// HandlerFunc defines the function signature for HTTP handler functions in rmhttp, as well as
// implementing the rmhttp.Handler interface.
//
// The only difference between a http.HandlerFunc and rmhttp.HandlerFunc is that our version
// can return errors. The signature is the same otherwise, so as to provide as familiar an
// API as possible.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

// ServeHTTP fulfills the http.Handler interface, and part of the rmhttp.Handler interface. It
// behaves exactly the same as a http.Handler.ServeHTTP call.
func (hf HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = hf(w, r)
}

// ServeHTTPWithError implements part of the rmhttp.Handler interface. It behaves very similarly
// to http.Handler.ServeHTTP, except that it also returns an error.
//
// It is also functionally equivalent to just calling the HandlerFunc directly.
func (hf HandlerFunc) ServeHTTPWithError(w http.ResponseWriter, r *http.Request) error {
	return hf(w, r)
}
