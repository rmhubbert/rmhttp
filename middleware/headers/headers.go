package headers

import "net/http"

// HeaderMiddleware creates and returns a MiddlewareFunc that will apply all of the headers
// that have been passed in.
func Middleware(headers map[string]string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for key, value := range headers {
				w.Header().Add(key, value)
			}
			next.ServeHTTP(w, r)
		})
	}
}
