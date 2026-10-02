package httplogger

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/grokify/mogo/log/sanitize"
)

// Middleware returns an HTTP middleware that logs each request via slog.
//
// The client IP recorded in the log is resolved with a trust-all policy: the leftmost usable
// X-Forwarded-For entry, then X-Real-IP, then the connection peer address. Candidates are
// validated and normalized with net/netip, so the field is always a plain IP address or empty, and
// an unparseable or absent client IP still produces exactly one log entry.
//
// Because proxy headers are trusted from any peer, the logged IP can be spoofed by clients unless
// you terminate behind a proxy you control. To only believe headers from known proxies, resolve
// the client IP in a trusted-proxy-aware middleware and log that value instead.
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The wrapper records the status and body size the handler produces. The clock runs from
			// before the wrap until the handler returns, so for a streaming or hijacked handler the
			// logged duration covers the whole connection rather than just the header write.
			start := time.Now()
			rec, metricsWriter := newMetricsWriter(w)

			next.ServeHTTP(metricsWriter, r)

			durationMs := time.Since(start).Milliseconds()
			code := rec.status
			written := rec.written

			// clientIP returns a validated "203.0.113.7"-style address or "", so it needs no further
			// sanitization: anything else it could have been is already rejected by net/netip.
			ip := clientIP(r)

			host := r.Header.Get("X-Forwarded-Host")
			if host == "" {
				host = r.Host
			}

			url := r.URL
			path := url.EscapedPath()
			if url.RawQuery != "" {
				path = path + "?" + url.RawQuery
			}

			// Every value a client controls is sanitized before it reaches the log (CWE-117 log
			// injection). strings.Map in sanitize.String returns the input unchanged when there is
			// nothing to strip, so this costs no allocations on clean input.
			// #nosec G706 - values are sanitized using github.com/grokify/mogo/log/sanitize
			sanitizedPath := sanitize.String(path)
			sanitizedReferer := sanitize.String(r.Referer())
			sanitizedAgent := sanitize.String(r.UserAgent())
			sanitizedHost := sanitize.String(host)
			sanitizedProto := sanitize.String(r.Proto)

			level := slog.LevelInfo
			if code >= http.StatusBadRequest {
				level = slog.LevelError
			}

			// Building the attributes in a stack array and handing them to LogAttrs allocates nothing
			// of its own: slog.String stores the string without copying it and integers are stored
			// inline, whereas a variadic slog.Info call would box every argument into an interface.
			// slog itself makes one allocation here, the attribute slice for the entries beyond the
			// five it can inline, and this record has eleven.
			//
			// The #nosec G706 pragmas in this handler suppress nothing today: gosec only flags user
			// data passed straight to a recognized logging call, and these values go through
			// slog.String instead. They stay on purpose — the sanitization they describe is
			// permanent, so when a future gosec learns to treat slog attribute values as
			// log-injection sinks, these pragmas pre-suppress exactly that false positive.
			// #nosec G706 - values are sanitized using github.com/grokify/mogo/log/sanitize
			attrs := [...]slog.Attr{
				slog.String("type", "http"),
				slog.Int("status", code),
				slog.String("ip", ip),
				slog.String("method", r.Method),
				slog.String("host", sanitizedHost),
				slog.String("path", sanitizedPath),
				slog.String("referer", sanitizedReferer),
				slog.String("ua", sanitizedAgent),
				slog.String("proto", sanitizedProto),
				slog.Int64("size", written),
				slog.Int64("duration", durationMs),
			}

			// The default logger is resolved per request on purpose: test suites and applications
			// retarget slog.Default(), and a logger captured when the middleware was built would keep
			// writing to the old destination.
			// #nosec G706 - values are sanitized using github.com/grokify/mogo/log/sanitize
			slog.Default().LogAttrs(r.Context(), level, http.StatusText(code), attrs[:]...)
		})
	}
}
