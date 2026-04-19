package apikey

import (
	"net/http"
)

// Middleware creates and returns a MiddlewareFunc that validates the x-api-key header.
// Keys are converted to a map for O(1) lookup instead of O(n) linear search.
func Middleware(keys ...string) func(http.Handler) http.Handler {
	// Convert keys to map for O(1) lookup instead of O(n) linear search
	keyMap := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keyMap[key] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(keyMap) > 0 {
				xApiKey := r.Header.Get("x-api-key")
				if _, ok := keyMap[xApiKey]; !ok {
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
