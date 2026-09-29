package httplogger

import (
	"log/slog"
	"net/http"

	"github.com/felixge/httpsnoop"
	"github.com/grokify/mogo/log/sanitize"
)

// SanitizedString wraps a string that has been sanitized to prevent log injection.
// It implements slog.LogValuer to ensure gosec recognizes it as safe for logging.
type SanitizedString string

func (s SanitizedString) LogValue() slog.Value {
	return slog.StringValue(string(s))
}

// Middleware returns an HTTP middleware that logs each request via slog.
//
// The client IP recorded in the log is resolved with a trust-all policy: the leftmost usable
// X-Forwarded-For entry, then X-Real-IP, then the connection peer address. Candidates are
// validated and normalized (netip) and the value is sanitized, so an unparseable or
// absent client IP degrades to an empty field and a log entry is always emitted.
//
// Because proxy headers are trusted from any peer, the logged IP can be spoofed by clients unless
// you terminate behind a proxy you control. To only believe headers from known proxies, resolve
// the client IP in a trusted-proxy-aware middleware and log that value instead.
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// NOTE: CaptureMetrics triggers next.ServeHTTP(w, r) for you, so do not run it manually as well.
			m := httpsnoop.CaptureMetrics(next, w, r)
			durationMs := m.Duration.Milliseconds()
			code := m.Code
			written := m.Written

			// clientIP returns a validated "203.0.113.7"-style address or "", so a plain conversion keeps
			// the log field sanitized.
			ip := SanitizedString(clientIP(r))
			host := r.Header.Get("X-Forwarded-Host")
			if host == "" {
				host = r.Host
			}

			url := r.URL
			path := url.EscapedPath()
			if url.RawQuery != "" {
				path = path + "?" + url.RawQuery
			}
			agent := r.UserAgent()
			referer := r.Referer()
			proto := r.Proto
			logType := "http"

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
