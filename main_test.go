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

// createHandlerFunc creates, initialises, and returns a rmhttp.HandlerFunc compatible function.
func createTestHandlerFunc(
	status int,
	body string,
	err error,
) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
		return err
	}
}

// createTestMiddlewareFunc creates, initialises and returns a rmhttp compatible middleware function
func createTestMiddlewareHandler(header string, value string) func(Handler) Handler {
	return func(next Handler) Handler {
		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			w.Header().Add(header, value)
			return next.ServeHTTPWithError(w, r)
		})
	}
}
