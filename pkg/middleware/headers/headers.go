package headers

import "net/http"

// HeaderPair represents a single header key-value pair for efficient iteration.
type HeaderPair struct {
	Key   string
	Value string
}

// Middleware creates and returns a MiddlewareFunc that will apply all of the headers
// that have been passed in. Headers are pre-converted to a slice for faster iteration
// and use Set() instead of Add() for single-value headers.
func Middleware(headers map[string]string) func(http.Handler) http.Handler {
	// Pre-convert map to slice at initialization time for faster iteration
	headerPairs := make([]HeaderPair, 0, len(headers))
	for key, value := range headers {
		headerPairs = append(headerPairs, HeaderPair{Key: key, Value: value})
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, pair := range headerPairs {
				w.Header().Set(pair.Key, pair.Value)
			}
			next.ServeHTTP(w, r)
		})
	}
}
