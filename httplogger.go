package rmhttp

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ------------------------------------------------------------------------------------------------
// HTTP LOGGER
// ------------------------------------------------------------------------------------------------

// HTTPLoggerMiddleware logs requests and errors using the passed Logger.
func HTTPLoggerMiddleware(logger Logger) MiddlewareFunc {
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
			hasErrorStatus := false

			if err != nil {
				if httpErr, ok := err.(HTTPError); ok {
					code = httpErr.Code
					message = httpErr.Error()
				} else if code < http.StatusBadRequest {
					code = http.StatusInternalServerError
					message = http.StatusText(http.StatusInternalServerError)
				}

				hasErrorStatus = true
			} else if code >= http.StatusBadRequest {
				hasErrorStatus = true
			}

			if hasErrorStatus {
				logger.Error(
					message,
					"method", r.Method,
					"host", host,
					"path", path,
					"status", code,
					"duration", duration,
				)

				return err
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
