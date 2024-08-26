package rmhttp

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"
)

// ------------------------------------------------------------------------------------------------
// ROUTE
// ------------------------------------------------------------------------------------------------

// A Route encapsulates all of the information that the router will need to satisfy an HTTP
// request. Alongside supplying standard information such as what HTTP method and URL
// pattern a handler should be bound to, the Route also allows the enclosed handler
// to be configured with their own timeout, headers, and middleware.
type Route struct {
	method     string
	pattern    string
	handler    Handler
	middleware []MiddlewareFunc
	timeout    Timeout
	headers    map[string]string
}

// NewRoute validates the input, then creates, initialises and returns a pointer to a Route. The
// validation step ensures that a valid HTTP method has been passed (http.MethodGet will be
// used, if not). The method will also be transformed to uppercase, and the pattern to
// lowercase.
func NewRoute(method string, pattern string, handler Handler) *Route {
	m := strings.ToUpper(method)
	if !slices.Contains(ValidHTTPMethods(), m) {
		method = http.MethodGet
	}
	return &Route{
		method:  method,
		pattern: strings.ToLower(pattern),
		handler: handler,
		headers: make(map[string]string),
		timeout: NewTimeout(10*time.Second, "Timeout"),
	}
}

// Use adds middleware handlers to the receiver Route.
//
// Each middleware handler will be wrapped to create a call stack with the order in which the
// middleware is added being maintained. So, for example, if the user added A and B
// middleware via this method, the resulting callstack would be as follows -
//
// Middleware A -> Middleware B -> Route Handler -> Middleware B -> Middleware A
//
// (This actually a slight simplification, as internal middleware such as HTTP Logging, CORS, HTTP
// Error Handling and Route Panic Recovery may also be inserted into the call stack, depending
// on how the App is configured).
//
// The middlewares argument is variadic, allowing the user to add multiple middleware functions
// in a single call.
//
// This method will return a pointer to the receiver Route, allowing the user to chain any of the
// other builder methods that Route implements.
func (route *Route) Use(middlewares ...func(Handler) Handler) *Route {
	for _, mw := range middlewares {
		route.middleware = append(route.middleware, MiddlewareFunc(mw))
	}
	return route
}

// Method returns the HTTP Method set for the Route as a string. It partially implements the
// Routable interface.
func (route *Route) Method() string {
	return route.method
}

// Pattern returns the URL pattern set for the Route as a string. It partially implements the
// Routable interface.
func (route *Route) Pattern() string {
	return route.pattern
}

// Handler returns the rmhttp Handler bound to the Route. It partially implements the Routable
// interface.
func (route *Route) Handler() Handler {
	return route.handler
}

// Headers returns the map of HTTP headers that have been added to the Route.
func (route *Route) Headers() map[string]string {
	return route.headers
}

// Timeout returns the Timeout object that has been added to the Route.
func (route *Route) Timeout() Timeout {
	return route.timeout
}

// Middleware returns the slice of MiddlewareFuncs that have been added to the Route.
func (route *Route) Middleware() []MiddlewareFunc {
	m := route.middleware
	headersMiddleware, ok := route.createHeaderMiddleware()
	if ok {
		m = append(m, headersMiddleware)
	}
	return m
}

// WithTimeout sets a request timeout amount for this route.
//
// This method will return a pointer to the receiver Route, allowing the user to chain any of the
// other builder methods that Route implements.
func (route *Route) WithTimeout(timeout time.Duration, message string) *Route {
	route.timeout = NewTimeout(timeout, message)
	return route
}

// WithHeader sets an HTTP header for this route.
//
// This method will return a pointer to the receiver Route, allowing the user to chain any of the
// other builder methods that Route implements.
func (route *Route) WithHeader(key, value string) *Route {
	route.headers[key] = value
	return route
}

// createHeaderMiddleware creates and returns a MiddlewareFunc that will apply all of the headers
// that have been added to the Route.
func (route *Route) createHeaderMiddleware() (MiddlewareFunc, bool) {
	if len(route.Headers()) > 0 {
		// Create simple middleware for adding the headers
		headersMiddleware := func(next Handler) Handler {
			return HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
				for key, value := range route.Headers() {
					w.Header().Add(key, value)
				}

				return next.ServeHTTPWithError(w, r)
			})
		}
		return MiddlewareFunc(headersMiddleware), true
	}
	return MiddlewareFunc(func(h Handler) Handler { return h }), false
}

// String is used internally to calculate a string signature for use as map keys, etc.
func (route *Route) String() string {
	return fmt.Sprint(route.method, " ", route.pattern)
}

// ------------------------------------------------------------------------------------------------
// ROUTE SERVICE
// ------------------------------------------------------------------------------------------------

// routeService supplies functionality for managing Route objects in the application. This
// includes providing interfaces for adding and removing routes, as well as applying route
// specific timeouts, middleware and headers.
type routeService struct {
	routes map[string]*Route
	logger Logger
}

// newRouteService creates, initialises, and then returns a pointer to a new routeService.
func newRouteService(logger Logger) *routeService {
	return &routeService{
		routes: make(map[string]*Route),
		logger: logger,
	}
}

// addRoute saves the passed Route object to an internal map, which will be used at server start
// to register all of the application routes with the router.
//
// This allows us to modify Routes prior to application start without causing the underlying
// http.ServeMux to throw an error.
func (rts *routeService) addRoute(route *Route) {
	rts.routes[route.String()] = route
}

// loadRoutes registers each Route with the passed Router
func (rts *routeService) loadRoutes(routes []*Route, router *Router) {
	for _, route := range routes {
		router.Handle(route)
	}
}
