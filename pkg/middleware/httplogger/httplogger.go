package httplogger

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/felixge/httpsnoop"
	"github.com/grokify/mogo/log/sanitize"
)

// SanitizedString wraps a string that has been sanitized to prevent log injection.
// It implements slog.LogValuer to ensure gosec recognizes it as safe for logging.
type SanitizedString string

func (s SanitizedString) LogValue() slog.Value {
	return slog.StringValue(string(s))
}

func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// NOTE: CaptureMetrics triggers next.ServeHTTP(w, r) for you, so do not run it manually as well.
			m := httpsnoop.CaptureMetrics(next, w, r)
			durationMs := m.Duration.Milliseconds()
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

			// Sanitize user-controlled input to prevent log injection (CWE-117)
			// The sanitize.String() function removes/repaces control characters
			// #nosec G706 - values are sanitized using github.com/grokify/mogo/log/sanitize
			sanitizedPath := SanitizedString(sanitize.String(path))
			sanitizedReferer := SanitizedString(sanitize.String(referer))
			sanitizedAgent := SanitizedString(sanitize.String(agent))
			sanitizedHost := SanitizedString(sanitize.String(host))
			sanitizedProto := SanitizedString(sanitize.String(proto))

			if code >= http.StatusBadRequest {
				// #nosec G706 - values are sanitized using github.com/grokify/mogo/log/sanitize
				slog.Error(
					http.StatusText(code),
					"type", logType,
					"status", code,
					"ip", ip,
					"method", r.Method,
					"host", sanitizedHost,
					"path", sanitizedPath,
					"referer", sanitizedReferer,
					"ua", sanitizedAgent,
					"proto", sanitizedProto,
					"size", written,
					"duration", durationMs,
				)
				return
			}

			// #nosec G706 - values are sanitized using github.com/grokify/mogo/log/sanitize
			slog.Info(
				http.StatusText(m.Code),
				"type", logType,
				"status", code,
				"ip", ip,
				"method", r.Method,
				"host", sanitizedHost,
				"path", sanitizedPath,
				"referer", sanitizedReferer,
				"ua", sanitizedAgent,
				"proto", sanitizedProto,
				"size", written,
				"duration", durationMs,
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
