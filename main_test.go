package rmhttp

import "net/http"

const (
	testAddress string = "localhost:8080"
)

// MockLogger implements the Logger interface but doesn't do any logging. It's only used
// in tests, where a logger needs passed to an initialising function, but we don't
// need to test the logging.
type MockLogger struct{}

func (ml MockLogger) Debug(string, ...any) {}
func (ml MockLogger) Info(string, ...any)  {}
func (ml MockLogger) Warn(string, ...any)  {}
func (ml MockLogger) Error(string, ...any) {}

// createTestHandlerFunc creates, initialises, and returns an http.HandlerFunc compatible function.
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
