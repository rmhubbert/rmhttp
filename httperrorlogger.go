package rmhttp

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ------------------------------------------------------------------------------------------------
// HTTP ERROR HANDLER
// ------------------------------------------------------------------------------------------------

// HTTPErrorHandlerMiddleware returns a MiddlwareFunc compatible function that handles any errors
// that have been returned by a handler. It will also create an appropriate HTTP error in the
// case of the response having a status code in the error range (400 and above), but no
// error was returned from the handler. This will allow any other middleware to assume
// that if they have not received an error, then no error has occurred.
func HTTPErrorLoggerMiddleware(logger Logger) MiddlewareFunc {
	return func(next Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			cw := NewCaptureWriter(w)
			defer cw.Persist()

			start := time.Now()
			err := next.ServeHTTPWithError(cw, r)
			duration := fmt.Sprintf("%vms", time.Since(start).Milliseconds())

			host := strings.Join(r.Header.Values("X-Forwarded-For"), ",")
			if host == "" {
				logger.Debug(
					fmt.Sprintf(
						"httplogger: no X-Forwarded-For header found, using Host header instead: %v",
						r.Host,
					),
				)
				host = r.Host
			}

			url := r.URL
			path := url.EscapedPath()
			query := url.RawQuery
			if query != "" {
				path += "?" + query
			}

			code := cw.Code
			message := cw.Body

			if err != nil || cw.Code > http.StatusBadRequest {
				if httpErr, ok := err.(HTTPError); ok {
					code = httpErr.Code
					message = httpErr.Err.Error()
				}

				logger.Error(
					message,
					"method", r.Method,
					"host", host,
					"path", path,
					"status", code,
					"duration", duration,
				)
			}

			logger.Info(
				http.StatusText(code),
				"method", r.Method,
				"host", host,
				"path", path,
				"status", code,
				"duration", duration,
			)

			return err
		})
	}
}
