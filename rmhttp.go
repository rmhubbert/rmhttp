// Package rmhttp implements a lightweight wrapper around the standard library web server provided
// by http.Server and http.ServeMux, and adds an intuitive fluent interface for easy use and
// configuration of route grouping, logging, CORS, panic recovery, header management, timeouts,
// and middleware.
//
// # Pattern Matching
//
// Routes use Go's net/http.ServeMux pattern matching with these features:
//
//	Path Parameters:
//	  - Use {param} for named path parameters (e.g., /users/{id})
//	  - Access via r.PathValue("param") in handlers
//
//	Wildcard Patterns:
//	  - Use {param...} for trailing wildcard parameters (e.g., /files/{path...})
//	  - Matches any remaining path segments
//
//	Method-Specific Routes:
//	  - Include method in pattern: POST /users/{id}
//	  - Method-specific routes take precedence over generic routes
//
//	Pattern Precedence:
//	  1. Exact match with method (e.g., GET /users)
//	  2. Exact match without method (e.g., /users)
//	  3. Wildcard match (e.g., /files/{path...})
//
// Examples:
//
//	app.Get("/users/{id}", handler)              // Path parameter
//	app.Post("/users/{id}", handler)             // Method-specific
//	app.Get("/files/{path...}", handler)         // Wildcard
//	app.Get("/api/{version}/users/{id}", handler) // Multiple params
//
// # Middleware
//
// Middleware functions are applied in the order they are added, with each
// middleware wrapping the next. The first middleware added runs first on
// requests and last on responses.
//
// Example:
//
//	app.Use(middleware1)  // Runs first on request, last on response
//	app.Use(middleware2)  // Runs second on request, second on response
//	app.Use(middleware3)  // Runs third on request, first on response
//
// Request flow: middleware1 → middleware2 → middleware3 → handler
// Response flow: handler → middleware3 → middleware2 → middleware1
//
// # Concurrency
//
// The App and Server types are designed to be used from multiple goroutines:
//
//   - Route registration (Get, Post, etc.) is safe for concurrent use
//   - Middleware registration is safe for concurrent use
//   - Server.Start() should be called from a single goroutine
//
// The underlying http.ServeMux is used for thread-safe route matching.
package rmhttp

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/rmhubbert/rmhttp/v5/pkg/middleware/headers"
)

// ------------------------------------------------------------------------------------------------
// APP
// ------------------------------------------------------------------------------------------------

// App encapsulates the application and provides the public API, as well as orchestrating the core
// library functionality.
type App struct {
	Server        *Server
	Router        *Router
	rootGroup     *Group
	errorHandlers map[int]http.Handler
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

	router := NewRouter()
	server := NewServer(
		config.Server,
		router,
	)
	server.maybeUpdateTimeout(time.Duration(config.Server.RequestTimeout) * time.Second)

	rootGroup := NewGroup("")
	// rootGroup.Timeout = NewTimeout(
	// 	time.Duration(config.Timeout.RequestTimeout)*time.Second,
	// 	config.Timeout.TimeoutMessage,
	// )

	errorHandlers := map[int]http.Handler{
		http.StatusNotFound:         createDefaultHandler(http.StatusNotFound),
		http.StatusMethodNotAllowed: createDefaultHandler(http.StatusMethodNotAllowed),
	}

	return &App{
		Server:        server,
		Router:        router,
		rootGroup:     rootGroup,
		errorHandlers: errorHandlers,
	}
}

// Handle binds the passed http.Handler to the specified route method and pattern.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) Handle(method string, pattern string, handler http.Handler) *Route {
	route := NewRoute(
		strings.TrimSpace(strings.ToUpper(method)),
		strings.TrimSpace(strings.ToLower(pattern)),
		handler,
	)
	app.rootGroup.Route(route)
	return route
}

// HandleFunc binds the passed http.HandlerFunc to the specified route method and pattern.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) HandleFunc(
	method string,
	pattern string,
	handlerFunc http.HandlerFunc,
) *Route {
	return app.Handle(method, pattern, http.HandlerFunc(handlerFunc))
}

// Get binds the passed handler to the specified route pattern for GET requests.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) Get(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Route {
	return app.HandleFunc(http.MethodGet, pattern, handlerFunc)
}

// Post binds the passed handler to the specified route pattern for POST requests.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) Post(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Route {
	return app.HandleFunc(http.MethodPost, pattern, handlerFunc)
}

// Put binds the passed handler to the specified route pattern for PUT requests.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) Put(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Route {
	return app.HandleFunc(http.MethodPut, pattern, handlerFunc)
}

// Patch binds the passed handler to the specified route pattern for PATCH requests.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) Patch(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Route {
	return app.HandleFunc(http.MethodPatch, pattern, handlerFunc)
}

// Delete binds the passed handler to the specified route pattern for DELETE requests.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) Delete(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Route {
	return app.HandleFunc(http.MethodDelete, pattern, handlerFunc)
}

// Options binds the passed handler to the specified route pattern for OPTIONS requests.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) Options(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Route {
	return app.HandleFunc(http.MethodOptions, pattern, handlerFunc)
}

// Static creates and binds a filesystem handler to the specified pattern for GET requests.
//
// If the pattern contains a trailing slash, the filesystem handler may not behave as expected.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) Static(pattern string, targetDir string) *Route {
	fsh := http.StripPrefix(pattern, http.FileServer(http.Dir(targetDir)))
	return app.HandleFunc(http.MethodGet, pattern, fsh.ServeHTTP)
}

// Redirect creates and binds a redirect handler to the specified pattern for GET requests.
//
// A temporary redirect status code will be used if the passed code is not in the 300 -
// 308 range.
//
// This method will return a pointer to the new Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (app *App) Redirect(pattern string, target string, code int) *Route {
	if code < http.StatusMultipleChoices || code > http.StatusPermanentRedirect {
		code = http.StatusTemporaryRedirect
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, code)
	})
	return app.Handle("GET", pattern, handler)
}

// StatusNotFoundHandler registers a handler to be used when an internal 404 error is thrown.
func (app *App) StatusNotFoundHandler(handler http.HandlerFunc) {
	app.errorHandlers[http.StatusNotFound] = http.HandlerFunc(handler)
}

// StatusMethodNotAllowedHandler registers a handler to be used when an internal 405 error is thrown.
func (app *App) StatusMethodNotAllowedHandler(
	handler http.HandlerFunc,
) {
	app.errorHandlers[http.StatusMethodNotAllowed] = http.HandlerFunc(handler)
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

// WithTimeout sets a request timeout amount and message at the global level.
//
// This method will return a pointer to the app, allowing the user to
// chain any of the other builder methods that the app implements.
func (app *App) WithTimeout(timeout time.Duration, message string) *App {
	app.rootGroup.WithTimeout(timeout, message)
	return app
}

// WithHeader adds an HTTP header at the global level. Calling this method more than once with the
// same key will overwrite the existing header.
//
// This method will return a pointer to the app, allowing the user to
// chain any of the other builder methods that the app implements.
func (app *App) WithHeader(key string, value string) *App {
	app.rootGroup.WithHeader(key, value)
	return app
}

// WithMiddleware is a convenience method for adding global middleware handlers.
//
// This method will return a pointer to the app, allowing the user to chain
// any of the other builder methods that the app implements.
func (app *App) WithMiddleware(middlewares ...func(http.Handler) http.Handler) *App {
	app.rootGroup.WithMiddleware(middlewares...)
	return app
}

// Use is a convenience method for adding global middleware handlers. It uses WithMiddleware
// behind the scenes.
//
// This method will return a pointer to the app, allowing the user to chain any of the
// other builder methods that the app implements.
func (app *App) Use(middlewares ...func(http.Handler) http.Handler) *App {
	app.rootGroup.WithMiddleware(middlewares...)
	return app
}

// Compile prepares the app for starting by applying the middleware, and loading the Routes. It
// should be the last function to be called before starting the Server.
func (app *App) Compile() {
	routes := app.rootGroup.ComputedRoutes()

	for _, route := range routes {
		middleware := []func(http.Handler) http.Handler{}

		if len(route.ComputedHeaders()) > 0 {
			middleware = append(middleware, headers.Middleware(route.ComputedHeaders()))
		}

		if len(route.ComputedMiddleware()) > 0 {
			middleware = append(middleware, route.ComputedMiddleware()...)
		}

		if timeout := route.ComputedTimeout(); timeout.Enabled {
			// Give the Server a chance to update it's TCP level timeout so that the connection doesn't
			// timeout before the request. It will only update if this timeout is longer than the
			// existing TCP timeout.
			app.Server.maybeUpdateTimeout(timeout.Duration)
			middleware = append(middleware, TimeoutMiddleware(route.ComputedTimeout()))
		}

		var handler = route.Handler
		if len(middleware) > 0 {
			handler = applyMiddleware(
				route.Handler,
				middleware,
			)
		}

		app.Router.Handle(route.Method, route.ComputedPattern(), handler)
	}

	// Add the error handlers to the router with any global middleware added.
	for code, errorHandler := range app.errorHandlers {
		app.Router.AddErrorHandler(code, applyMiddleware(errorHandler, app.rootGroup.Middleware))
	}
}

// ListenAndServe compiles and loads the registered Routes, and then starts the Server without SSL.
func (app *App) ListenAndServe() error {
	app.Compile()
	return app.Server.ListenAndServe()
}

// ListenAndServeTLS compiles and loads the registered Routes, and then starts the Server with the
// SSL certificate and key at the file paths passed as the arguments.
func (app *App) ListenAndServeTLS(cert string, key string) error {
	app.Compile()
	return app.Server.ListenAndServeTLS(cert, key)
}

// Shutdown stops the Server.
func (app *App) Shutdown(ctx context.Context) error {
	return app.Server.Shutdown(ctx)
}
