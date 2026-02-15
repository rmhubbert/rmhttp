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
	Method     string
	Pattern    string
	Handler    http.Handler
	Middleware []func(http.Handler) http.Handler
	Timeout    Timeout
	Headers    map[string]string
	Parent     *Group
}

// NewRoute validates the input, then creates, initialises and returns a pointer to a Route. The
// validation step ensures that a valid HTTP method has been passed (http.MethodGet will be
// used, if not). The method will also be transformed to uppercase, and the pattern to
// lowercase.
func NewRoute(method string, pattern string, handler http.Handler) *Route {
	m := strings.ToUpper(method)
	if !slices.Contains(ValidHTTPMethods(), m) {
		method = http.MethodGet
	}
	if err := validatePattern(pattern); err != nil {
		panic(fmt.Sprintf("invalid route pattern: %v", err))
	}
	return &Route{
		Method:  method,
		Pattern: strings.ToLower(pattern),
		Handler: handler,
		Headers: make(map[string]string),
	}
}

// ComputedPattern dynamically calculates the pattern for the Route. It returns the URL pattern as a
// string.
func (route *Route) ComputedPattern() string {
	return route.buildPattern(route.Pattern, route.Parent)
}

// buildPattern builds a URL pattern by conatenating any parent Group patterns together with the
// Route pattern.
func (route *Route) buildPattern(pattern string, parent *Group) string {
	if parent == nil {
		return pattern
	}
	pattern = fmt.Sprintf("%s%s", parent.Pattern, pattern)
	return route.buildPattern(pattern, parent.Parent)
}

// ComputedHeaders dynamically calculates the HTTP headers that have been added to the Route and
// any parent Groups.
func (route *Route) ComputedHeaders() map[string]string {
	return route.findHeaders(route.Headers, route.Parent)
}

// findHeaders collects all of the headers set on the Route, plus any parent groups.
func (route *Route) findHeaders(headers map[string]string, parent *Group) map[string]string {
	if parent == nil {
		return headers
	}
	// Only add a parent header if it hasn't already been set in the child.
	for key, value := range parent.Headers {
		if _, ok := headers[key]; !ok {
			headers[key] = value
		}
	}
	return route.findHeaders(headers, parent.Parent)
}

// Timeout returns the Timeout object that has been added to the Route.
func (route *Route) ComputedTimeout() Timeout {
	if !route.Timeout.Enabled {
		return route.findEnabledTimeout(route.Parent)
	}
	return route.Timeout
}

// findEnabledTimeout searches for an enabled Timeout in any parent Group.
func (route *Route) findEnabledTimeout(parent *Group) Timeout {
	if parent == nil {
		return Timeout{}
	}
	if parent.Timeout.Enabled {
		return parent.Timeout
	}
	return route.findEnabledTimeout(parent.Parent)
}

// Middleware returns the slice of MiddlewareFuncs that have been added to the Route.
func (route *Route) ComputedMiddleware() []func(http.Handler) http.Handler {
	return route.findMiddleware(route.Parent, route.Middleware)
}

// findMiddleware searches for any middleware in any parent Group.
func (route *Route) findMiddleware(
	parent *Group,
	middleware []func(http.Handler) http.Handler,
) []func(http.Handler) http.Handler {
	if parent == nil {
		return middleware
	}
	middleware = append(parent.Middleware, middleware...)
	return route.findMiddleware(parent.Parent, middleware)
}

// WithMiddleware adds Middleware handlers to the receiver Route.
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
func (route *Route) WithMiddleware(middlewares ...func(http.Handler) http.Handler) *Route {
	route.Middleware = append(route.Middleware, middlewares...)
	return route
}

// Use is a convenience method for adding middleware handlers to a Route. It uses WithMiddleware
// behind the scenes.
//
// This method will return a pointer to the receiver Route, allowing the user to chain any of the
// other builder methods that Route implements.
func (route *Route) Use(middlewares ...func(http.Handler) http.Handler) *Route {
	return route.WithMiddleware(middlewares...)
}

// WithTimeout sets a request timeout amount and message for this route.
//
// This method will return a pointer to the receiver Route, allowing the user to chain any of the
// other builder methods that Route implements.
func (route *Route) WithTimeout(timeout time.Duration, message string) *Route {
	route.Timeout = NewTimeout(timeout, message)
	return route
}

// WithHeader sets an HTTP header for this Route. Calling this method with the same key more than
// once will overwrite the existing header.
//
// This method will return a pointer to the receiver Route, allowing the user to chain any of the
// other builder methods that Route implements.
func (route *Route) WithHeader(key, value string) *Route {
	route.Headers[key] = value
	return route
}

// String is used internally to calculate a string signature for use as map keys, etc.
func (route *Route) String() string {
	return fmt.Sprint(route.Method, " ", route.Pattern)
}

// validatePattern checks if a route pattern is valid
func validatePattern(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("route pattern cannot be empty")
	}

	if len(pattern) > 256 {
		return fmt.Errorf("route pattern too long (max 256 chars): %s", pattern)
	}

	// Check for balanced braces in path parameters
	openBraces := strings.Count(pattern, "{")
	closeBraces := strings.Count(pattern, "}")
	if openBraces != closeBraces {
		return fmt.Errorf("unbalanced braces in pattern: %s", pattern)
	}

	// Check for invalid characters (basic validation)
	// Allow alphanumeric, /, -, _, ., {, }, and ...
	for _, r := range pattern {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') &&
			r != '/' && r != '-' && r != '_' && r != '.' && r != '{' && r != '}' {
			return fmt.Errorf("invalid character '%c' in pattern: %s", r, pattern)
		}
	}

	return nil
}
