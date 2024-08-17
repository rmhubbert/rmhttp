package rmhttp

import "net/http"

type NetHTTPApp struct {
	app *App
}

func NewNetHTTP(c ...Config) *NetHTTPApp {
	config := Config{}
	if len(c) > 0 {
		config = c[0]
	}

	return &NetHTTPApp{
		app: New(config),
	}
}

// HandleFunc converts the passed handler function to a rmhttp.HandlerFunc, and then binds the
// newly converted HandlerFunc to the specified route method and pattern.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (nha *NetHTTPApp) HandleFunc(method string, pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) *Route {
	return nha.app.HandleFunc(method, pattern, ConvertHandlerFunc(handlerFunc))
}

// Handler to the specified route method and pattern.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (nha *NetHTTPApp) Handle(method string, pattern string, handler http.Handler) *Route {
	return nha.app.Handle(method, pattern, ConvertHandler(handler))
}
