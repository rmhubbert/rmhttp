package rmhttp

import "net/http"

// validHTTPMethods returns slice of strings containing all of the HTTP methods
// that rmhttp will accept.
func validHTTPMethods() []string {
	return []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}
}
