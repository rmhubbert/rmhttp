package rmhttp

import "net/http"

// Handler extends the http.Handler interface with ServeHTTPWithError, allowing Handlers to return errors.
type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
	ServeHTTPWithError(http.ResponseWriter, *http.Request) error
}

// handlerFunc defines the function signature for HTTP handler functions in rmhttp.
//
// the only difference between a http.HandlerFunc and rmhttp.HandlerFunc is that our
// version can return errors. The signature is the same otherwise, so as to provide
// as familiar an API as possible.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

// ServeHTTP fulfills the http.Handler interface, so that we can substitute http.Handlers
// with HandlerFuncs, if necessary.
func (hf HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = hf(w, r)
}

// ServeHTTPWithError is only really here to fulfill the Handler interface, so that
// we can substitute HandlerFuncs with Handlers, if necessary.
//
// It is functionally equivalent to just calling the HandlerFunc directly.
func (hf HandlerFunc) ServeHTTPWithError(w http.ResponseWriter, r *http.Request) error {
	return hf(w, r)
}
