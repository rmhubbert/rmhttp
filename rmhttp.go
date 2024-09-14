// Package rmhttp implements a lightweight wrapper around the standard library web server provided
// by http.Server and http.ServeMux, and adds an intuitive fluent interface for easy use and
// configuration of route grouping, centralised error handling, logging, CORS, panic
// recovery, SSL configuration, header management, timeouts and middleware.
//
// The package allows you to use either standard http.Handler functions, or rmhttp.Handler
// functions, which are identical to http.Handler functions, with the addition of
// returning an error. Returning an error from your handlers allows rmhttp to
// provide centralised error handling, but if you'd rather handle your
// errors in your handler, you can simply use the net/http
// compayible App, and use net/http handlers natively.
package rmhttp

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// ------------------------------------------------------------------------------------------------
// APP
// ------------------------------------------------------------------------------------------------

// App encapsulates the application and provides the public API, as well as orchestrating the core
// library functionality.
type App struct {
	logger           Logger
	Server           *Server
	Router           *Router
	rootGroup        *Group
	errorStatusCodes map[error]int
}

// New creates, initialises and returns a pointer to a new App. An optional configuration can be
// passed to configure many parts of the system, such as cors, SSL, and timeouts.
//
// If you chose not to pass in a configuration, rmhttp will first attempt to load configuration
// values from environment variables, and if they're not found, will apply sensible defaults.
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
	serverConfig := ServerConfig{
		TimeoutConfig: config.Timeout,
		SSLConfig:     config.SSL,
		Host:          config.Host,
		Port:          config.Port,
	}
	server := NewServer(
		serverConfig,
		router,
		config.Logger,
	)

	rootGroup := NewGroup("")
	rootGroup.Timeout = NewTimeout(
		time.Duration(config.Timeout.RequestTimeout)*time.Second,
		config.Timeout.TimeoutMessage,
	)

	return &App{
		Server:           server,
		Router:           router,
		logger:           config.Logger,
		rootGroup:        rootGroup,
		errorStatusCodes: make(map[error]int),
	}
}

// Handle binds the passed rmhttp.Handler to the specified route method and pattern.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) Handle(method string, pattern string, handler Handler) *Route {
	route := NewRoute(
		strings.TrimSpace(strings.ToUpper(method)),
		strings.TrimSpace(strings.ToLower(pattern)),
		handler,
	)
	app.rootGroup.Route(route)
	return route
}

// HandleFunc converts the passed handler function to a rmhttp.HandlerFunc, and then binds it to
// the specified route method and pattern.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) HandleFunc(
	method string,
	pattern string,
	handlerFunc func(http.ResponseWriter, *http.Request) error,
) *Route {
	return app.Handle(method, pattern, HandlerFunc(handlerFunc))
}

// Group creates, initialises, and returns a pointer to a Route Group.
//
// This is typically used to create new Routes as part of the Group, but can also be used to add
// Group specific middleware, timeouts, etc.
//
// This method will return a pointer to the new Group, allowing the user to chain any of the other
// builder methods that Group implements.
func (app *App) Group(pattern string) *Group {
	group := NewGroup(pattern)
	app.rootGroup.Group(group)
	return group
}

// Routes returns a map of the currently added Routes.
func (app *App) Routes() map[string]*Route {
	return app.rootGroup.ComputedRoutes()
}

// Route adds a Route to the application at the top level.
//
// This allows us to overwrite Routes prior to application start without causing the underlying
// http.ServeMux to throw an error.
func (app *App) Route(route *Route) {
	app.rootGroup.Route(route)
}

// Compile prepares the app for starting by applying the middleware, and loading the Routes. It
// should be the last function to be called before starting the Server.
func (app *App) Compile() {
	routes := app.rootGroup.ComputedRoutes()

	for _, route := range routes {
		middleware := []MiddlewareFunc{
			HTTPErrorLoggerMiddleware(app.logger),
			HeaderMiddleware(route.ComputedHeaders()),
		}
		middleware = append(middleware, route.ComputedMiddleware()...)
		middleware = append(middleware, HTTPErrorHandlerMiddleware(app.errorStatusCodes))

		if timeout := route.ComputedTimeout(); timeout.Enabled {
			// Give the Server a chance to update it's TCP level timeout so that the connection doesn't
			// timeout before the request. It will only update if this timeout is longer than the
			// existing TCP timeout.
			app.Server.maybeUpdateTimeout(timeout.Duration)
			middleware = append(middleware, TimeoutMiddleware(route.ComputedTimeout()))
		}

		handler := applyMiddleware(
			route.Handler,
			middleware,
		)

		app.Router.Handle(route.Method, route.Pattern, handler)
	}

	// app.Router.loadRoutes(routes)
}

// ListenAndServe compiles and loads the registered Routes, and then starts the Server without SSL.
func (app *App) ListenAndServe() error {
	app.Compile()
	return app.Server.ListenAndServe()
}

// ListenAndServeTLS compiles and loads the registered Routes, and then starts the Server with SSL.
func (app *App) ListenAndServeTLS() error {
	app.Compile()
	return app.Server.ListenAndServeTLS()
}

// Start compiles and loads the registered Routes, and then starts the Server with graceful
// shutdown management.
func (app *App) Start() error {
	app.Compile()
	return app.Server.Start(false)
}

// Shutdown stops the Server.
func (app *App) Shutdown() error {
	return app.Server.Shutdown(context.Background())
}
