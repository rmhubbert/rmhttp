package httplogger

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/felixge/httpsnoop"
)

func Middleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// NOTE: CaptureMetrics triggers next.ServeHTTP(w, r) for you, so do not run it manually as well.
			m := httpsnoop.CaptureMetrics(next, w, r)
			duration := fmt.Sprintf("%dms", m.Duration.Milliseconds())
			code := m.Code
			written := m.Written

			ip := realIp(r)
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
			agent := r.UserAgent()
			referer := r.Referer()
			proto := r.Proto
			logType := "http"

			if code >= http.StatusBadRequest {
				logger.Error(
					http.StatusText(code),
					"type", logType,
					"status", code,
					"ip", ip,
					"method", r.Method,
					"host", host,
					"path", path,
					"referer", referer,
					"ua", agent,
					"proto", proto,
					"size", written,
					"duration", duration,
				)
				return
			}

			logger.Info(
				http.StatusText(m.Code),
				"type", logType,
				"status", code,
				"ip", ip,
				"method", r.Method,
				"host", host,
				"path", path,
				"referer", referer,
				"ua", agent,
				"proto", proto,
				"size", written,
				"duration", duration,
			)
		})
	}
}

// Request.RemoteAddress contains the port, which is not desired.
func removePort(ra string) string {
	index := strings.LastIndex(ra, ":")
	if index == -1 {
		return ra
	}
	return ra[:index]
}

// requestGetRemoteAddress returns ip address of the client making the request,
// taking into account http proxies
func realIp(r *http.Request) string {
	header := r.Header
	xRealIP := header.Get("X-Real-Ip")
	xForwardedFor := header.Get("X-Forwarded-For")
	if xRealIP == "" && xForwardedFor == "" {
		return removePort(r.RemoteAddr)
	}
	if xForwardedFor != "" {
		parts := strings.Split(xForwardedFor, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		return parts[0]
	}
	return xRealIP
}
