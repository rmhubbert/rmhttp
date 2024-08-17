package rmhttp

import (
	"fmt"
	"strings"
)

// A Route encapsulates all of the information that the router will need to satisfy an HTTP request.
// Alongside supplying standard information such as what HTTP method and URL pattern a handler
// should be bound to, the Route also allows the enclosed handler to be configured with
// their own timeout, headers, and middleware.
type Route struct {
	method  string
	pattern string
	handler Handler
}

// String implements the Stringer interface. It is used internally to calculate a string signature
// for use as map keys, etc.
func (route *Route) String() string {
	return fmt.Sprint(strings.ToUpper(route.method), " ", strings.ToLower(route.pattern))
}
