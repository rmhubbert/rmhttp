package rmhttp

import (
	"bytes"
	"log/slog"
	"net/http"
	"os"
	"testing"
)

const (
	testAddress string = "localhost:8080"
)

var out = &bytes.Buffer{}

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(out, nil)))
	exitCode := m.Run()
	os.Exit(exitCode)
}

// creaeTestHandlerFunc creates, initialises, and returns an http.HandlerFunc compatible function.
func createTestHandlerFunc(
	status int,
	body string,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

// createTestMiddlewareFunc creates, initialises and returns a compatible middleware function
func createTestMiddlewareHandler(header string, value string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add(header, value)
			next.ServeHTTP(w, r)
		})
	}
}
