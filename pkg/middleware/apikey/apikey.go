package apikey

import (
	"net/http"
	"slices"
)

func Middleware(keys ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(keys) > 0 {
				xApiKey := r.Header.Get("x-api-key")
				if xApiKey == "" || !slices.Contains(keys, xApiKey) {
					http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
