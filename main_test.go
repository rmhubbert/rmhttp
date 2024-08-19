package rmhttp

import "net/http"

// createHandlerFunc creates, initialises, and returns a rmhttp.HandlerFunc compatible function.
func createTestHandlerFunc(
	status int,
	body string,
	err error,
) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(status)
		w.Write([]byte(body))
		return err
	}
}

// createNetHTTPHandlerFunc creates, initialises, and returns a http.HandlerFunc compatible
// function.
func createTestNetHTTPHandlerFunc(
	status int,
	body string,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}
}
