package rmhttp

import "net/http"

// ------------------------------------------------------------------------------------------------
// PACKAGE CONSTANTS AND FUNCTIONS THAT RETURN COLLECTIONS OF CONSTANTS
// ------------------------------------------------------------------------------------------------
// ValidHTTPMethods returns a slice of strings containing all of the HTTP methods that rmhttp will
// accept.
func ValidHTTPMethods() []string {
	return []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
	}
}
