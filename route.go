package rmhttp

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// ------------------------------------------------------------------------------------------------
// ROUTABLE INTERFACE
// ------------------------------------------------------------------------------------------------
// The Routable interface allows any type that implements it to be used as a route within rmhttp.
type Routable interface {
	Method() string
	Pattern() string
	Handler() Handler
	fmt.Stringer
	Usable
}

// ------------------------------------------------------------------------------------------------
// ROUTE
// ------------------------------------------------------------------------------------------------
// A Route encapsulates all of the information that the router will need to satisfy an HTTP
// request. Alongside supplying standard information such as what HTTP method and URL
// pattern a handler should be bound to, the Route also allows the enclosed handler
// to be configured with their own timeout, headers, and middleware.
//
// Route implements the Routable, Usable and standard library Stringer interfaces.
type Route struct {
	method     string
	pattern    string
	handler    Handler
	middleware []func(Handler) Handler
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
	}
}

// Method implements part of the Routable interface. It returns the Route's method.
func (route *Route) Method() string {
	return route.method
}

// Pattern implements part of the Routable interface. It returns the Route's pattern.
func (route *Route) Pattern() string {
	return route.pattern
}

// Handler implements part of the Routable & Usable interfaces. It returns the Route's handler.
func (route *Route) Handler() Handler {
	return route.handler
}

// Middleware implements part of the Usable interface. It returns the Route's middleware.
func (route *Route) Middleware() []func(Handler) Handler {
	return route.middleware
}

// Use adds middleware handlers to the receiver Route.
//
// Each middleware handler will be wrapped to create a call stack with the order in which the
// middleware is added being maintained. So, for example, if the user added GZIP and
// Sentry middleware via this method, the resulting callstack would be as follows -
//
// Middleware 1 -> Middleware 2 -> Route Handler -> Middleware 2 -> Middleware 1
//
// (This actually a slight simplication, as internal middleware such as HTTP Logging, CORS,
// HTTP Error Handling and Route Panic Recovery may also be inserted into the call stack, depending
// on how the App is configured).
//
// The middlewareFuncs argument is variadic, allowing the user to add multiple
// middleware functions in a single call.
//
// This method will return a pointer to the receiver Route, allowing the user to chain
// any of the other builder methods that Route implements.
func (route *Route) Use(middlewareFuncs ...func(Handler) Handler) *Route {
	route.middleware = append(route.middleware, middlewareFuncs...)
	return route
}

// String implements the Stringer interface. It is used internally to calculate a string signature
// for use as map keys, etc.
func (route *Route) String() string {
	return fmt.Sprint(route.Method(), " ", route.Pattern())
}

// ------------------------------------------------------------------------------------------------
// ROUTE SERVICE
// ------------------------------------------------------------------------------------------------
// routeService supplies functionality for managing Routable objects in the application. This
// includes providing interfaces for adding and removing routes, as well as applying route
// specific timeouts, middleware and headers.
type routeService struct {
	router *Router
	routes map[string]Routable
}

// newRouteService creates, initialises, and then returns a pointer to a new routeService.
func newRouteService(router *Router) *routeService {
	return &routeService{
		router: router,
		routes: make(map[string]Routable),
	}
}

// addRoute saves the passed Routable object to an internal map, which will be used at server start
// to register all of the application routes with the router.
//
// This allows us to modify Routes prior to application start without causing the underlying
// http.ServeMux to throw an error.
func (rts *routeService) addRoute(route Routable) {
	rts.routes[route.String()] = route
}

// compileRoutes applies middleware, timeouts and headers to each registered route before passing
// them to the Router, which in turn registers each Route with the underlying http.ServeMux
func (rts *routeService) compileRoutes() {
	for _, route := range rts.routes {
		rts.router.Handle(route)
	}
}
