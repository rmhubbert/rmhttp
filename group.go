package rmhttp

import (
	"net/http"
	"strings"
	"time"
)

// ------------------------------------------------------------------------------------------------
// GROUP
// ------------------------------------------------------------------------------------------------

// A Group allows for grouping sub groups or routes under a route prefix. It also enables you to
// add headers, timeout and middleware once to every sub group and route included in the group.
type Group struct {
	Pattern    string
	Middleware []func(http.Handler) http.Handler
	Timeout    Timeout
	Headers    map[string]string
	Parent     *Group
	Routes     map[string]*Route
	Groups     map[string]*Group
}

// NewGroup creates, initialises, and returns a pointer to a new Group
func NewGroup(pattern string) *Group {
	return &Group{
		Pattern: pattern,
		Headers: make(map[string]string),
		Routes:  make(map[string]*Route),
		Groups:  make(map[string]*Group),
	}
}

// Handle binds the passed rmhttp.Handler to the specified route method and pattern.
//
// This method will return a pointer to the receiver Group, allowing the user to
// chain any of the other builder methods that Group implements.
func (group *Group) Handle(method string, pattern string, handler http.Handler) *Group {
	route := NewRoute(
		strings.TrimSpace(strings.ToUpper(method)),
		strings.TrimSpace(strings.ToLower(pattern)),
		handler,
	)
	return group.Route(route)
}

// HandleFunc converts the passed handler function to a rmhttp.HandlerFunc, and then binds it to
// the specified route method and pattern.
//
// This method will return a pointer to the receiver Group, allowing the user to chain
// any of the other builder methods that Group implements.
func (group *Group) HandleFunc(
	method string,
	pattern string,
	handlerFunc http.HandlerFunc,
) *Group {
	return group.Handle(method, pattern, http.HandlerFunc(handlerFunc))
}

// Get binds the passed handler to the specified route pattern for GET requests.
//
// This method will return a pointer to the receiver Group, allowing the user to chain
// any of the other builder methods that Group implements.
func (group *Group) Get(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Group {
	return group.HandleFunc(http.MethodGet, pattern, handlerFunc)
}

// Post binds the passed handler to the specified route pattern for POST requests.
//
// This method will return a pointer to the receiver Group, allowing the user to chain
// any of the other builder methods that Group implements.
func (group *Group) Post(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Group {
	return group.HandleFunc(http.MethodPost, pattern, handlerFunc)
}

// Put binds the passed handler to the specified route pattern for PUT requests.
//
// This method will return a pointer to the receiver Group, allowing the user to chain
// any of the other builder methods that Group implements.
func (group *Group) Put(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Group {
	return group.HandleFunc(http.MethodPut, pattern, handlerFunc)
}

// Patch binds the passed handler to the specified route pattern for PATCH requests.
//
// This method will return a pointer to the receiver Group, allowing the user to chain
// any of the other builder methods that Group implements.
func (group *Group) Patch(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Group {
	return group.HandleFunc(http.MethodPatch, pattern, handlerFunc)
}

// Delete binds the passed handler to the specified route pattern for DELETE requests.
//
// This method will return a pointer to the receiver Group, allowing the user to chain
// any of the other builder methods that Group implements.
func (group *Group) Delete(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Group {
	return group.HandleFunc(http.MethodDelete, pattern, handlerFunc)
}

// Options binds the passed handler to the specified route pattern for OPTIONS requests.
//
// This method will return a pointer to the receiver Group, allowing the user to chain
// any of the other builder methods that Group implements.
func (group *Group) Options(
	pattern string,
	handlerFunc http.HandlerFunc,
) *Group {
	return group.HandleFunc(http.MethodOptions, pattern, handlerFunc)
}

// Route adds the passed Route to this Group.
//
// This method will return a pointer to the receiver Group, allowing the user to chain any of the
// other builder methods that Group implements.
func (group *Group) Route(route *Route) *Group {
	group.Routes[route.String()] = route
	route.Parent = group
	return group
}

// Group adds the passed Group as a sub group to this Group.
//
// This method will return a pointer to the receiver Group, allowing the user to chain any of the
// other builder methods that Group implements.
func (group *Group) Group(g *Group) *Group {
	group.Groups[group.Pattern] = g
	g.Parent = group
	return group
}

// WithMiddleware adds Middleware handlers to the receiver Group.
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
// This method will return a pointer to the receiver Group, allowing the user to chain any of the
// other builder methods that Group implements.
func (group *Group) WithMiddleware(middlewares ...func(http.Handler) http.Handler) *Group {
	group.Middleware = append(group.Middleware, middlewares...)
	return group
}

// Use is a convenience method for adding middleware handlers to a Group. It uses WithMiddleware
// behind the scenes.
//
// This method will return a pointer to the receiver Group, allowing the user to chain any of the
// other builder methods that Group implements.
func (group *Group) Use(middlewares ...func(http.Handler) http.Handler) *Group {
	return group.WithMiddleware(middlewares...)
}

// WithHeader sets an HTTP header for this Group. Calling this method with the same key more than
// once will overwrite the existing header.
//
// This method will return a pointer to the receiver Group, allowing the user to chain any of the
// other builder methods that Group implements.
func (group *Group) WithHeader(key, value string) *Group {
	group.Headers[key] = value
	return group
}

// WithTimeout sets a request timeout amount and message for this Group.
//
// This method will return a pointer to the receiver Group, allowing the user to chain any of the
// other builder methods that Group implements.
func (group *Group) WithTimeout(timeout time.Duration, message string) *Group {
	group.Timeout = NewTimeout(timeout, message)
	return group
}

// ComputedRoutes returns a map of unique Routes composed from this Group and any sub Groups of
// this Group.
func (group *Group) ComputedRoutes() map[string]*Route {
	routes := make(map[string]*Route)
	findUniqueRoutes(routes, group)
	return routes
}

// findUniqueRoutes recursively creates a map of unique Routes from this Group and any sub Groups
// of this Group.
func findUniqueRoutes(routes map[string]*Route, g *Group) {
	if g == nil {
		return
	}

	for key, route := range g.Routes {
		if _, ok := routes[key]; !ok {
			routes[key] = route
		}
	}

	for _, subGroup := range g.Groups {
		findUniqueRoutes(routes, subGroup)
	}
}
