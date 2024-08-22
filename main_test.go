package rmhttp

import "net/http"

const (
	testAddress string = "localhost:8080"
)

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
// func createTestMiddlewareHandler(header string, value string) func(Handler) Handler {
// 	return func(next Handler) Handler {
// 		return HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
// 			w.Header().Add(header, value)
// 			return next.ServeHTTPWithError(w, r)
// 		})
// 	}
// }
