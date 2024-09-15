package rmhttp

import "net/http"

// HeaderMiddleware creates and returns a MiddlewareFunc that will apply all of the headers
// that have been passed in.
func HeaderMiddleware(headers map[string]string) MiddlewareFunc {
	return func(next Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			for key, value := range headers {
				w.Header().Add(key, value)
			}
			return next.ServeHTTPWithError(w, r)
		})
	}
}
