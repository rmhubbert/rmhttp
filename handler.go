package rmhttp

import "net/http"

// createDefaultHandler creates and returns an http.HandlerFunc that simply sets the response status to
// the passed code, and response body to the textual version of the same code.
//
// It is generally used to create default error handlers.
func createDefaultHandler(code int) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(http.StatusText(code)))
	})
}
