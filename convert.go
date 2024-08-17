package rmhttp

import "net/http"

// ConvertHandlerFunc converts, then returns, the passed Net/HTTP compatible HandlerFunc
// function to one that fulfils the rmhttp.HandlerFunc signature
func ConvertHandlerFunc(handlerFunc func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		handlerFunc(w, r)
		return nil
	}
}

// ConvertHandler converts, then returns, the passed Net/HTTP Handler to a rmhttp.HandlerFunc,
// which fulfils the rmhttp.Handler interface.
func ConvertHandler(handler http.Handler) HandlerFunc {
	return HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		handler.ServeHTTP(w, r)
		return nil
	})
}
