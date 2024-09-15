package rmhttp

import (
	"errors"
	"net/http"
)

// ------------------------------------------------------------------------------------------------
// HTTP ERROR HANDLER
// ------------------------------------------------------------------------------------------------

// HTTPErrorHandlerMiddleware returns a MiddlwareFunc compatible function that handles any errors
// that have been returned by a handler. It will also create an appropriate HTTP error in the
// case of the response having a status code in the error range (400 and above), but no
// error was returned from the handler. This will allow any other middleware to assume
// that if they have not received an error, then no error has occurred.
func HTTPErrorHandlerMiddleware(registeredErrors map[error]int) MiddlewareFunc {
	return func(next Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			cw := NewCaptureWriter(w)
			err := next.ServeHTTPWithError(cw, r)
			if err == nil {
				cw.Persist()
				// It's possible that a handler wrote an error state to the response writer, but did not
				// return an error. This conditional should catch that and create an appropriate HTTP
				// error.
				if cw.Code >= http.StatusBadRequest {
					return NewHTTPError(errors.New(cw.Body), cw.Code)
				}
				return nil
			}

			// Check to see if we've been passed an HTTP error.
			if httpErr, ok := err.(HTTPError); ok {
				Error(w, httpErr.Error(), httpErr.Code)
				return httpErr
			}

			// Check against all of the registed errors.
			if len(registeredErrors) > 0 {
				for e, code := range registeredErrors {
					if errors.Is(err, e) {
						Error(w, err.Error(), code)
						return NewHTTPError(err, code)
					}
				}
			}

			// Check to see if an error code has been set in the response. If so, we can transform the
			// error into an HTTPError with the correct status code.
			if cw.Code >= http.StatusBadRequest {
				Error(w, err.Error(), cw.Code)
				return NewHTTPError(err, cw.Code)
			}

			// If we get here, then we haven't been able to identify the error that was returned from the
			// next handler. Return a generic HTTP 500 error.
			Error(w, err.Error(), http.StatusInternalServerError)
			return NewHTTPError(err, http.StatusInternalServerError)
		})
	}
}
