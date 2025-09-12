package rmhttp

import (
	"fmt"
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
	return fmt.Sprintf("%d: %s", e.Code, e.Err.Error())
}
