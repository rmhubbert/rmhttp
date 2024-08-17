// Package rmhttp implements a lightweight wrapper around the Go standard library
// web server provided by http.Server and http.ServeMux, and adds an intuitive
// fluent interface for easy use and configuration of route grouping,
// centralised error handling, logging, CORS, panic recovery,
// SSL configuration, header management, timeouts and
// middleware.
package rmhttp

import (
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// App encapsulates the application and provides the public API, as well as orchestrating
// the core library functionality.
type App struct {
	Server *Server
	Router *Router
	routes map[string]*Route
}

// New initialises and returns a new instance of rmhttp. An optional configuration can
// be passed to configure many parts of the system, such as cors, SSL, and timeouts.
//
// If you chose not to pass in a configuration, rmhttp will first attempt to load
// configuration values from environment variables, and if they're not found,
// will apply sensible defaults.
func New(c ...Config) *App {
	cfg := Config{}
	if len(c) > 0 {
		cfg = c[0]
	}
	config, err := LoadConfig(cfg)
	if err != nil {
		panic("cannot load config")
	}

	// If a custom logger hasn't been passed through the config, create one with
	// sensible defaults.
	if config.Logger == nil {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		if config.Debug {
			logger.Debug("creating slog logger with loglevel set to DEBUG")
			logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
		}
		logger.Debug("setting logger as default")
		slog.SetDefault(logger)
		config.Logger = logger
	}

	router := NewRouter(config.Logger)
	server := NewServer(router, config.Host, config.Port, config.SSL.Cert, config.SSL.Key, config.Logger)

	return &App{
		Server: server,
		Router: router,
		routes: make(map[string]*Route),
	}
}

// Handle binds the passed rmhttp.Handler to the specified route method and pattern.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) Handle(method string, pattern string, handler Handler) *Route {
	route := &Route{
		method:  strings.TrimSpace(strings.ToUpper(method)),
		pattern: strings.TrimSpace(strings.ToLower(pattern)),
		handler: handler,
	}
	app.addRoute(route)
	return route
}

// HandleFunc converst the passed handler function to a rmhttp.HandlerFunc, and then binds it to
// the specified route method and pattern.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) HandleFunc(method string, pattern string, handlerFunc func(http.ResponseWriter, *http.Request) error) *Route {
	return app.Handle(method, pattern, HandlerFunc(handlerFunc))
}

// addRoute saves the passed Route pointer to an internal map, which will be used at server start
// to register all of the application routes with the router.
//
// This allows us to overwrite Routes prior to application start without causing the underlying
// http.ServeMux to throw an error.
func (app *App) addRoute(route *Route) {
	app.routes[route.String()] = route
}

// loadRoutes passes each registered route to the Router, which in turn registers each Route
// with the underlying http.ServeMux
func (app *App) loadRoutes() {
	for _, route := range app.routes {
		app.Router.handle(route)
	}
}

// Start loads the registered routes, and then starts the Server with graceful shutdown management.
func (app *App) Start() error {
	app.loadRoutes()
	return app.Server.Start(false)
}
