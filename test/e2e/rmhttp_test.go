package e2e

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rmhubbert/rmhttp"
	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// APP E2E TESTS
// ------------------------------------------------------------------------------------------------

// Test_Handler tests binding an rmhttp.Handler to a method & pattern with a selection of route
// patterns, methods and status codes.
func Test_Handle(t *testing.T) {
	// Set up the App
	app := rmhttp.New()
	defer func() {
		_ = app.Shutdown()
	}()
	// Add handlers for all of our tests
	for _, test := range handlerTests {
		route := app.Handle(
			test.method,
			test.pattern,
			rmhttp.HandlerFunc(createTestHandlerFunc(test.status, test.body, test.err)),
		)
		assert.Equal(
			t,
			fmt.Sprintf("%s %s", strings.ToUpper(test.method), strings.ToLower(test.pattern)),
			route.String(),
		)
	}
	// Start the App and wait for it to be responsive
	startServer(app)

	// Run our tests
	for _, test := range handlerTests {
		t.Run(test.name, func(t *testing.T) {
			url := fmt.Sprintf("http://%s%s", testAddress, test.pathToTest)
			req, err := http.NewRequest(test.method, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("get request failed: %v", err)
			}

			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, test.body, string(body), "they should be equal")
			assert.Equal(t, test.status, res.StatusCode, "they should be equal")
		})
	}
}

// Test_HandlerFunc tests binding an rmhttp.Handler to a method & pattern with a selection of route
// patterns, methods and status codes.
func Test_HandleFunc(t *testing.T) {
	// Set up the App
	app := rmhttp.New()
	defer func() {
		_ = app.Shutdown()
	}()

	// Add handlers for all of our tests
	for _, test := range handlerTests {
		route := app.HandleFunc(
			test.method,
			test.pattern,
			createTestHandlerFunc(test.status, test.body, test.err),
		)
		assert.Equal(
			t,
			fmt.Sprintf("%s %s", strings.ToUpper(test.method), strings.ToLower(test.pattern)),
			route.String(),
		)
	}

	// Start the App and wait for it to be responsive
	startServer(app)

	// Run our tests
	for _, test := range handlerTests {
		t.Run(test.name, func(t *testing.T) {
			url := fmt.Sprintf("http://%s%s", testAddress, test.pathToTest)
			req, err := http.NewRequest(test.method, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("get request failed: %v", err)
			}

			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, test.body, string(body), "they should be equal")
			assert.Equal(t, test.status, res.StatusCode, "they should be equal")
		})
	}
}

// Test_Route tests route handling with pre-created Routes.
func Test_Route(t *testing.T) {
	// Set up the App
	app := rmhttp.New()
	defer func() {
		_ = app.Shutdown()
	}()

	// Add handlers for all of our tests
	for _, test := range handlerTests {
		route := rmhttp.NewRoute(
			test.method,
			test.pattern,
			rmhttp.HandlerFunc(createTestHandlerFunc(test.status, test.body, test.err)),
		)
		app.Route(route)

		expectedPattern := fmt.Sprintf("%s %s", test.method, test.pattern)
		assert.Equal(
			t,
			expectedPattern,
			fmt.Sprintf("%s %s", test.method, route.ComputedPattern()),
		)
	}

	// Start the App and wait for it to be responsive
	startServer(app)

	// Run our tests
	for _, test := range handlerTests {
		t.Run(test.name, func(t *testing.T) {
			url := fmt.Sprintf("http://%s%s", testAddress, test.pattern)
			req, err := http.NewRequest(test.method, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("get request failed: %v", err)
			}

			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, test.body, string(body), "they should be equal")
			assert.Equal(t, test.status, res.StatusCode, "they should be equal")
		})
	}
}

// Test_Group tests route handling within groups.
func Test_Group(t *testing.T) {
	// Set up the App
	app := rmhttp.New()
	defer func() {
		_ = app.Shutdown()
	}()

	group := app.Group("/group")
	// Add handlers for all of our tests
	for _, test := range handlerTests {
		route := rmhttp.NewRoute(
			test.method,
			test.pattern,
			rmhttp.HandlerFunc(createTestHandlerFunc(test.status, test.body, test.err)),
		)
		group.Route(route)

		expectedPattern := fmt.Sprintf("%s %s%s", test.method, group.Pattern, test.pattern)
		assert.Equal(
			t,
			expectedPattern,
			fmt.Sprintf("%s %s", test.method, route.ComputedPattern()),
		)
	}

	// Start the App and wait for it to be responsive
	startServer(app)

	// Run our tests
	for _, test := range handlerTests {
		t.Run(test.name, func(t *testing.T) {
			groupedPattern := fmt.Sprintf("%s%s", group.Pattern, test.pathToTest)
			url := fmt.Sprintf("http://%s%s", testAddress, groupedPattern)
			req, err := http.NewRequest(test.method, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("get request failed: %v", err)
			}

			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, test.body, string(body), "they should be equal")
			assert.Equal(t, test.status, res.StatusCode, "they should be equal")
		})
	}
}

// Test_Convenience_Handlers tests the convenience route handlers
func Test_Convenience_Handlers(t *testing.T) {
	// Set up the App
	app := rmhttp.New()
	defer func() {
		_ = app.Shutdown()
	}()

	tests := []struct {
		name    string
		method  string
		pattern string
		handler func(string, func(http.ResponseWriter, *http.Request) error) *rmhttp.Route
	}{
		{"registers a GET handler", http.MethodGet, "/get", app.Get},
		{"registers a POST handler", http.MethodPost, "/post", app.Post},
		{"registers a PATCH handler", http.MethodPatch, "/patch", app.Patch},
		{"registers a PUT handler", http.MethodPut, "/put", app.Put},
		{"registers a DELETE handler", http.MethodDelete, "/delete", app.Delete},
		{"registers a OPTIONS handler", http.MethodOptions, "/options", app.Options},
	}

	// We need to register our routes before starting the server.
	for _, test := range tests {
		content := fmt.Sprintf("%s content", test.method)
		route := test.handler(test.pattern, createTestHandlerFunc(http.StatusOK, content, nil))
		assert.IsType(t, &rmhttp.Route{}, route, "it should be of this type")
		assert.Equal(t, test.pattern, route.ComputedPattern(), "they should be equal")
	}

	// Start the App and wait for it to be responsive
	startServer(app)

	// Run our tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content := fmt.Sprintf("%s content", test.method)

			url := fmt.Sprintf("http://%s%s", testAddress, test.pattern)
			req, err := http.NewRequest(test.method, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("get request failed: %v", err)
			}

			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, content, string(body), "they should be equal")
			assert.Equal(t, http.StatusOK, res.StatusCode, "they should be equal")
		})
	}
}
