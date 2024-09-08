package rmhttp

import "time"

// ------------------------------------------------------------------------------------------------
// GROUP
// ------------------------------------------------------------------------------------------------

// A Group allows for grouping sub groups or routes under a route prefix. It also enables you to
// add headers, timeout and middleware once to every sub group and route included in the group.
type Group struct {
	Pattern    string
	Middleware []MiddlewareFunc
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
func (group *Group) WithMiddleware(middlewares ...func(Handler) Handler) *Group {
	for _, mw := range middlewares {
		group.Middleware = append(group.Middleware, MiddlewareFunc(mw))
	}
	return group
}

// Use is a convenience method for adding middleware handlers to a Group. It uses WithMiddleware
// behind the scenes.
//
// This method will return a pointer to the receiver Group, allowing the user to chain any of the
// other builder methods that Group implements.
func (group *Group) Use(middlewares ...func(Handler) Handler) *Group {
	return group.WithMiddleware(middlewares...)
}

// WithHeader sets an HTTP header for this Group. Calling this method more than once will either
// overwrite an existing header, or add a new one.
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
