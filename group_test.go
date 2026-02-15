package rmhttp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
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
		http.HandlerFunc(createTestHandlerFunc(200, "test body")),
	)
	routes := group.ComputedRoutes()
	assert.Len(t, routes, 1, "they should be equal")

	expectedKey := fmt.Sprintf("GET %s", fmt.Sprintf("%s%s", groupPattern, handlerPattern))
	if route, ok := routes[expectedKey]; !ok {
		t.Errorf("route not found: %s", expectedKey)
	} else {
		assert.Equal(t, "GET", route.Method, "they should be equal")
		assert.Equal(
			t,
			fmt.Sprintf("%s%s", groupPattern, handlerPattern),
			route.ComputedPattern(),
			"they should be equal",
		)
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
		createTestHandlerFunc(200, "test body"),
	)
	routes := group.ComputedRoutes()
	assert.Len(t, routes, 1, "they should be equal")

	expectedKey := fmt.Sprintf("GET %s", fmt.Sprintf("%s%s", groupPattern, handlerPattern))
	if route, ok := routes[expectedKey]; !ok {
		t.Errorf("route not found: %s", expectedKey)
	} else {
		assert.Equal(t, "GET", route.Method, "they should be equal")
		assert.Equal(
			t,
			fmt.Sprintf("%s%s", groupPattern, handlerPattern),
			route.ComputedPattern(),
			"they should be equal",
		)
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
		handler func(string, http.HandlerFunc) *Group
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
			test.handler(handlerPattern, createTestHandlerFunc(200, "test body"))

			routes := group.ComputedRoutes()

			expectedKey := fmt.Sprintf("%s %s", test.method,
				fmt.Sprintf("%s%s", groupPattern, handlerPattern))
			if route, ok := routes[expectedKey]; !ok {
				t.Errorf("route not found: %s", expectedKey)
			} else {
				assert.Equal(t, test.method, route.Method, "they should be equal")
				assert.Equal(
					t,
					fmt.Sprintf("%s%s", groupPattern, handlerPattern),
					route.ComputedPattern(),
					"they should be equal",
				)
				assert.NotNil(t, route.Handler, "it should not be nil")
			}
		})
	}
}

// Test_Group_PatternCollision checks that routes with the same pattern in different groups don't collide.
func Test_Group_PatternCollision(t *testing.T) {
	app := New()

	group1 := app.Group("/api")
	group1.Get("/users", createTestHandlerFunc(200, "api users"))

	group2 := app.Group("/admin")
	group2.Get("/users", createTestHandlerFunc(200, "admin users"))

	routes := app.Routes()

	// Verify both routes exist without collision
	assert.Len(t, routes, 2)

	// Compile the app to load routes into the router
	app.Compile()

	// Verify each route returns the correct handler
	req1 := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	w1 := httptest.NewRecorder()
	app.Router.ServeHTTP(w1, req1)
	assert.Equal(t, "api users", w1.Body.String())

	req2 := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	w2 := httptest.NewRecorder()
	app.Router.ServeHTTP(w2, req2)
	assert.Equal(t, "admin users", w2.Body.String())
}
