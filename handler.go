package rmhttp

import "net/http"

// createDefaultHandler creates and returns an http.HandlerFunc that simply sets the response status
// to the passed code, and response body to the textual version of the same code.
//
// The body is computed once, at creation time, and reused for every response: the error handlers
// this produces run on the request-serving hot path, and converting http.StatusText on every call
// would allocate per response. All ResponseWriter implementations copy the slice passed to Write,
// so sharing it across concurrent requests is safe as long as no handler mutates it.
//
// It is generally used to create default error handlers.
func createDefaultHandler(code int) http.HandlerFunc {
	body := []byte(http.StatusText(code))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		_, _ = w.Write(body)
	})
}
