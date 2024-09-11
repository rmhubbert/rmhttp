package rmhttp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// An HTTPError represents an error with an additional HTTP status code
type HTTPError struct {
	Err  error
	Code int
}

// NewHTTPError creates and returns a new, initialised pointer to a HTTPError
func NewHTTPError(err error, code int) HTTPError {
	return HTTPError{
		Err:  err,
		Code: code,
	}
}

// Unwrap returns the underlying Error that this HTTPError wraps.
//
// This method allows an HTTPError to be used by errors.Is().
func (e HTTPError) Unwrap() error {
	return e.Err
}

// Error returns the error text of the receiver HTTPError as a string.
//
// This method allows HTTPError to implement the standard library Error interface.
func (e HTTPError) Error() string {
	return fmt.Sprintf("error %d: %s", e.Code, e.Err.Error())
}

// Error is designed as a drop in replacement for http.Error.
//
// This function will check for a Content-Type header in the Response and create
// either a JSON or plain text error in the Response via the ResponseWriter.
//
// Plain text errors will default internally to being created with the
// http.Error function.
func Error(w http.ResponseWriter, body string, code int) {
	isJSON := false
	contentType := w.Header().Get("Content-Type")
	if contentType != "" {
		if strings.Contains(strings.ToLower(contentType), "application/json") ||
			strings.Contains(strings.ToLower(contentType), "application/vnd.api+json") {
			isJSON = true
		}
	}

	if isJSON {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(_HTTPErrorJSONResponseGenerator(code, body))
	} else {
		http.Error(w, body, code)
	}
}

// _HTTPErrorJSONResponseObject is a global variable that holds a function that will be used
// to create a struct that represents an HTTPError JSON response.
//
// We create a default implementation here, but the idea is that the user will be able to
// supply their own generator function via the App when initialising the system.
var _HTTPErrorJSONResponseGenerator = func(code int, message string) any {
	return struct {
		Err struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}{
		Err: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: message,
		},
	}
}
