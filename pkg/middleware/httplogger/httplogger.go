package httplogger

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/rmhubbert/rmhttp/pkg/capturewriter"
)

func Middleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cw := capturewriter.New(w)

			start := time.Now()
			next.ServeHTTP(cw, r)
			duration := fmt.Sprintf("%vms", time.Since(start).Milliseconds())

			host := strings.Join(r.Header.Values("X-Forwarded-For"), ",")
			if host == "" {
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

			var hasErrorStatus = false
			if code >= http.StatusBadRequest {
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
				return
			}

			logger.Info(
				http.StatusText(code),
				"method", r.Method,
				"host", host,
				"path", path,
				"status", code,
				"duration", duration,
			)
		})
	}
}
