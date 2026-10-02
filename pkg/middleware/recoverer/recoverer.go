// Package recoverer provides middleware that recovers panics raised while a request is
// being served, so one bad handler cannot take down the connection's flow, and turns
// them into a logged error plus a 500 response.
package recoverer

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/grokify/mogo/log/sanitize"
)

// Middleware creates and returns a MiddlewareFunc that recovers any panic raised by the
// handlers it wraps.
//
// A recovered panic would otherwise vanish: net/http's own recovery sits above this one,
// so once the panic is caught here the server never learns what went wrong. The panic
// value and the goroutine's stack are therefore logged via slog at Error level, and the
// client receives a bare 500 — unless the request was a protocol upgrade, where a status
// write would corrupt the tunneled connection.
//
// One value is deliberately not swallowed: http.ErrAbortHandler is the standard library's
// "drop this client, quietly" signal, and only the server itself can act on it (it closes
// the connection and logs nothing). Panics carrying it are re-raised so they reach the
// server's recovery. A bare panic(nil) arrives as a *runtime.PanicNilError since Go 1.21
// and is logged and answered like any other panic.
func Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rvr := recover()
				if rvr == nil {
					return
				}

				if rvr == http.ErrAbortHandler {
					// Return the abort to net/http: it closes the connection without
					// logging, which is the entire purpose of this sentinel value.
					panic(rvr)
				}

				// The default logger is resolved per request (as httplogger does) so
				// callers that retarget slog.Default(), tests included, are honored.
				// Panic values can embed client-controlled text, so every string that
				// reaches the log is sanitized against log injection (CWE-117); the
				// stack is included because its header line quotes the panic value.
				slog.Default().Error("panic recovered",
					"type", "http",
					"method", r.Method,
					"path", sanitize.String(r.URL.Path),
					"panic", sanitize.String(fmt.Sprintf("%v", rvr)),
					"stack", sanitize.String(string(debug.Stack())),
				)

				if r.Header.Get("Connection") != "Upgrade" {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
