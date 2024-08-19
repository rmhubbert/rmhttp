package rmhttp

import "net/http"

// ------------------------------------------------------------------------------------------------
// NET/HTTP COMPATIBLE APP
// ------------------------------------------------------------------------------------------------
// NetHTTPApp wraps rmhttp.App to supply net/http compatibility, allowing the user to use
// http.Handler compatible handlers for creating routes and middleware.
type NetHTTPApp struct {
	App
}

// NewNetHTTP creates, initialises and returns a pointer to a new App. An optional configuration
// can be passed to configure many parts of the system, such as cors, SSL, and timeouts.
//
// If you chose not to pass in a configuration, rmhttp will first attempt to load configuration
// values from environment variables, and if they're not found, will apply sensible defaults.
func NewNetHTTP(c ...Config) *NetHTTPApp {
	config := Config{}
	if len(c) > 0 {
		config = c[0]
	}

	return &NetHTTPApp{
		App: *New(config),
	}
}

// HandleFunc converts the passed http.HandleFunc compatible function to a rmhttp.HandlerFunc,
// and then binds the newly converted HandlerFunc to the specified route method and pattern.
//
// This method will return a pointer to the new Route, allowing the user to chain any of the
// other builder methods that Route implements.
func (nha *NetHTTPApp) HandleFunc(
	method string,
	pattern string,
	handlerFunc func(http.ResponseWriter, *http.Request),
) *Route {
	return nha.App.HandleFunc(method, pattern, ConvertHandlerFunc(handlerFunc))
}

// Handler converts the passed http.Handler toa rmhttp.Handler, and then binds the newly converted
// Handler to the specified route method and pattern.
//
// This method will return a pointer to the new Route, allowing the user to chain any of the
// other builder methods that Route implements.
func (nha *NetHTTPApp) Handle(method string, pattern string, handler http.Handler) *Route {
	return nha.App.Handle(method, pattern, ConvertHandler(handler))
}
