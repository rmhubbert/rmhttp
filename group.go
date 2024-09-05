package rmhttp

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
	return group
}
