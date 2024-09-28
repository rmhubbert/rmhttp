package rmhttp

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// GROUP TESTS
// ------------------------------------------------------------------------------------------------

// Test_Group_Handle checks that a handler can be successfully added to a Group
func Test_Group_Handle(t *testing.T) {
	groupPattern := "/group"
	handlerPattern := "/pattern"
	group := NewGroup(groupPattern)
	group.Handle(
		"get",
		handlerPattern,
		HandlerFunc(createTestHandlerFunc(200, "test body", nil)),
	)
	routes := group.ComputedRoutes()
	assert.Len(t, routes, 1, "they should be equal")

	expectedKey := fmt.Sprintf("GET %s", handlerPattern)
	if route, ok := routes[expectedKey]; !ok {
		t.Errorf("route not found: %s", expectedKey)
	} else {
		assert.Equal(t, "GET", route.Method, "they should be equal")
		assert.Equal(t, fmt.Sprintf("%s%s", groupPattern, handlerPattern), route.ComputedPattern(), "they should be equal")
		assert.NotNil(t, route.Handler, "it should not be nil")
	}
}

// Test_Group_HandleFunc checks that a handlerFunc can be successfully added to a Group
func Test_Group_HandleFunc(t *testing.T) {
	groupPattern := "/group"
	handlerPattern := "/pattern"
	group := NewGroup(groupPattern)
	group.HandleFunc(
		"get",
		handlerPattern,
		createTestHandlerFunc(200, "test body", nil),
	)
	routes := group.ComputedRoutes()
	assert.Len(t, routes, 1, "they should be equal")

	expectedKey := fmt.Sprintf("GET %s", handlerPattern)
	if route, ok := routes[expectedKey]; !ok {
		t.Errorf("route not found: %s", expectedKey)
	} else {
		assert.Equal(t, "GET", route.Method, "they should be equal")
		assert.Equal(t, fmt.Sprintf("%s%s", groupPattern, handlerPattern), route.ComputedPattern(), "they should be equal")
		assert.NotNil(t, route.Handler, "it should not be nil")
	}
}

// Test_Group_Convenience_Handlers checks that a handlerFunc can be successfully added to the Group with
// any of the convenience methods.
func Test_Group_Convenience_Handlers(t *testing.T) {
	groupPattern := "/group"
	handlerPattern := "/pattern"
	group := NewGroup(groupPattern)

	tests := []struct {
		name    string
		method  string
		handler func(string, func(http.ResponseWriter, *http.Request) error) *Group
	}{
		{"Get creates a route and adds it to the group with a GET method", "GET", group.Get},
		{"Post creates a route and adds it to the group with a Post method", "POST", group.Post},
		{
			"Patch creates a route and adds it to the group with a Patch method",
			"PATCH",
			group.Patch,
		},
		{"Put creates a route and adds it to the group with a Put method", "PUT", group.Put},
		{
			"Delete creates a route and adds it to the group with a Delete method",
			"DELETE",
			group.Delete,
		},
		{
			"Options creates a route and adds it to the group with a Options method",
			"OPTIONS",
			group.Options,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.handler(handlerPattern, createTestHandlerFunc(200, "test body", nil))

			routes := group.ComputedRoutes()

			expectedKey := fmt.Sprintf("%s %s", test.method, handlerPattern)
			if route, ok := routes[expectedKey]; !ok {
				t.Errorf("route not found: %s", expectedKey)
			} else {
				assert.Equal(t, test.method, route.Method, "they should be equal")
				assert.Equal(t, fmt.Sprintf("%s%s", groupPattern, handlerPattern), route.ComputedPattern(), "they should be equal")
				assert.NotNil(t, route.Handler, "it should not be nil")
			}
		})
	}
}
