package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/rmhubbert/rmhttp/v5"
	"github.com/rmhubbert/rmhttp/v5/pkg/middleware/recoverer"
	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// APP E2E TESTS
// ------------------------------------------------------------------------------------------------

var config = rmhttp.Config{
	Server: rmhttp.ServerConfig{
		Port: defaultPort,
	},
}

// Test_Handler tests binding an rmhttp.Handler to a method & pattern with a selection of route
// patterns, methods and status codes.
func Test_Handle(t *testing.T) {
	// Set up the App
	app := rmhttp.New(config)
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	// Add handlers for all of our tests
	for _, test := range handlerTests {
		route := app.Handle(
			test.method,
			test.pattern,
			http.HandlerFunc(createTestHandlerFunc(test.status, test.body)),
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

			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

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
	app := rmhttp.New(config)
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	// Add handlers for all of our tests
	for _, test := range handlerTests {
		route := app.HandleFunc(
			test.method,
			test.pattern,
			createTestHandlerFunc(test.status, test.body),
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

			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

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
	app := rmhttp.New(config)
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	// Add handlers for all of our tests
	for _, test := range handlerTests {
		route := rmhttp.NewRoute(
			test.method,
			test.pattern,
			http.HandlerFunc(createTestHandlerFunc(test.status, test.body)),
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

			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

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
	app := rmhttp.New(config)
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	group := app.Group("/group")
	// Add handlers for all of our tests
	for _, test := range handlerTests {
		route := rmhttp.NewRoute(
			test.method,
			test.pattern,
			http.HandlerFunc(createTestHandlerFunc(test.status, test.body)),
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

			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

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
	app := rmhttp.New(config)
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	tests := []struct {
		name    string
		method  string
		pattern string
		handler func(string, http.HandlerFunc) *rmhttp.Route
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
		route := test.handler(test.pattern, createTestHandlerFunc(http.StatusOK, content))
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

			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, content, string(body), "they should be equal")
			assert.Equal(t, http.StatusOK, res.StatusCode, "they should be equal")
		})
	}
}

// Test_Error_Handlers tests being able to register and trigger custom error handlers
func Test_Error_Handlers(t *testing.T) {
	// Set up the App
	app := rmhttp.New(config)
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	tests := []struct {
		name    string
		pattern string
		code    int
		method  string
		handler func(http.HandlerFunc)
	}{
		{
			"registers a 404 error handler",
			"/404",
			http.StatusNotFound,
			http.MethodGet,
			app.StatusNotFoundHandler,
		},
		{
			"registers a 405 error handler",
			"/405",
			http.StatusMethodNotAllowed,
			http.MethodPost,
			app.StatusMethodNotAllowedHandler,
		},
	}

	// We need to register our routes before starting the server.
	for _, test := range tests {
		content := fmt.Sprintf("%d content", test.code)
		test.handler(createTestHandlerFunc(test.code, content))
	}

	// We need a known route to test the 405 handler against.
	app.Get("/405", createTestHandlerFunc(http.StatusOK, "405 content"))

	// Start the App and wait for it to be responsive
	startServer(app)

	// Run our tests
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content := fmt.Sprintf("%d content", test.code)

			url := fmt.Sprintf("http://%s%s", testAddress, test.pattern)
			req, err := http.NewRequest(test.method, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("get request failed: %v", err)
			}

			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, content, string(body), "they should be equal")
			assert.Equal(t, test.code, res.StatusCode, "they should be equal")
		})
	}
}

// Test_Route_With_Headers tests that headers can be added to routes
func Test_Route_With_Headers(t *testing.T) {
	// Set up the App
	app := rmhttp.New(config)
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	// Add handlers for all of our tests
	for _, test := range handlerTests {
		route := rmhttp.NewRoute(
			test.method,
			test.pattern,
			http.HandlerFunc(createTestHandlerFunc(test.status, test.body)),
		)
		route.WithHeader("x-key", "value")
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

			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, test.body, string(body), "they should be equal")
			assert.Equal(t, test.status, res.StatusCode, "they should be equal")
			assert.Equal(t, "value", res.Header.Get("x-key"), "they should be equal")
		})
	}
}

// ------------------------------------------------------------------------------------------------
// PATH VALUE E2E TESTS
// ------------------------------------------------------------------------------------------------

// createPathValueHandlerFunc creates a handler that extracts path parameters and returns them in the response body.
func createPathValueHandlerFunc(paramNames ...string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		values := make([]string, 0, len(paramNames))
		for _, name := range paramNames {
			value := r.PathValue(name)
			values = append(values, fmt.Sprintf("%s=%s", name, value))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(strings.Join(values, "|")))
	}
}

// Test_PathValue_ExtractsParameters tests that dynamic path parameters are correctly extracted
// and made available via r.PathValue() in handlers.
func Test_PathValue_ExtractsParameters(t *testing.T) {
	// Set up the App
	app := rmhttp.New(config)
	app.Use(recoverer.Middleware())
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	tests := []struct {
		name          string
		method        string
		pattern       string
		pathToTest    string
		expectedParam string
	}{
		{
			name:          "extracts single path parameter",
			method:        http.MethodGet,
			pattern:       "/users/{id}",
			pathToTest:    "/users/123",
			expectedParam: "id=123",
		},
		{
			name:          "extracts string parameter with hyphens",
			method:        http.MethodGet,
			pattern:       "/posts/{slug}",
			pathToTest:    "/posts/hello-world",
			expectedParam: "slug=hello-world",
		},
		{
			name:          "extracts two path parameters",
			method:        http.MethodGet,
			pattern:       "/users/{user_id}/posts/{post_id}",
			pathToTest:    "/users/42/posts/100",
			expectedParam: "user_id=42|post_id=100",
		},
		{
			name:          "extracts parameter with special characters",
			method:        http.MethodGet,
			pattern:       "/files/{filename}",
			pathToTest:    "/files/test-file.txt",
			expectedParam: "filename=test-file.txt",
		},
		{
			name:          "extracts encoded parameter (spaces)",
			method:        http.MethodGet,
			pattern:       "/search/{query}",
			pathToTest:    "/search/hello%20world",
			expectedParam: "query=hello world",
		},
		{
			name:          "extracts UUID parameter",
			method:        http.MethodGet,
			pattern:       "/items/{uuid}",
			pathToTest:    "/items/550e8400-e29b-41d4-a716-446655440000",
			expectedParam: "uuid=550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:          "extracts wildcard path parameter",
			method:        http.MethodGet,
			pattern:       "/files/{path...}",
			pathToTest:    "/files/docs/readme.md",
			expectedParam: "path=docs/readme.md",
		},
		{
			name:          "extracts parameter from method-specific route",
			method:        http.MethodPost,
			pattern:       "POST /api/users/{id}",
			pathToTest:    "/api/users/999",
			expectedParam: "id=999",
		},
		{
			name:          "extracts parameter from group route",
			method:        http.MethodGet,
			pattern:       "/api/v1/group-users/{id}",
			pathToTest:    "/api/v1/group-users/777",
			expectedParam: "id=777",
		},
	}

	// Register handlers for all test cases (except group route which is handled separately)
	app.HandleFunc(http.MethodGet, "/users/{id}", createPathValueHandlerFunc("id"))
	app.HandleFunc(http.MethodGet, "/posts/{slug}", createPathValueHandlerFunc("slug"))
	app.HandleFunc(
		http.MethodGet,
		"/users/{user_id}/posts/{post_id}",
		createPathValueHandlerFunc("user_id", "post_id"),
	)
	app.HandleFunc(http.MethodGet, "/files/{filename}", createPathValueHandlerFunc("filename"))
	app.HandleFunc(http.MethodGet, "/search/{query}", createPathValueHandlerFunc("query"))
	app.HandleFunc(http.MethodGet, "/items/{uuid}", createPathValueHandlerFunc("uuid"))
	app.HandleFunc(http.MethodGet, "/files/{path...}", createPathValueHandlerFunc("path"))
	app.HandleFunc(http.MethodPost, "/api/users/{id}", createPathValueHandlerFunc("id"))

	// Register group example - this route should have the group prefix
	// Note: We use a different pattern for the group route to avoid key collision
	v1 := app.Group("/api/v1")
	v1.HandleFunc(http.MethodGet, "/group-users/{id}", createPathValueHandlerFunc("id"))

	// Start the App and wait for it to be responsive
	startServer(app)

	// Run our tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := fmt.Sprintf("http://%s%s", testAddress, tt.pathToTest)
			req, err := http.NewRequest(tt.method, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Errorf("get request failed: %v", err)
			}

			defer func() {
				err := res.Body.Close()
				if err != nil {
					t.Errorf("failed to close response body: %v", err)
				}
			}()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Errorf("failed to read response body: %v", err)
			}

			assert.Equal(t, http.StatusOK, res.StatusCode, "status code should be OK")
			assert.Equal(
				t,
				tt.expectedParam,
				string(body),
				"path parameter should be extracted correctly",
			)
		})
	}
}

// Test_PathValue_MethodSpecificPatterns tests that method-specific patterns work correctly
// and that the method-specific handler takes precedence over generic handlers.
func Test_PathValue_MethodSpecificPatterns(t *testing.T) {
	// Set up the App
	app := rmhttp.New(config)
	defer func() {
		_ = app.Shutdown(context.Background())
	}()

	// Register both generic and method-specific patterns
	app.HandleFunc(http.MethodGet, "/{resource}", createPathValueHandlerFunc("resource"))

	// Method-specific pattern should take precedence for DELETE
	app.HandleFunc(http.MethodDelete, "/{resource}", func(w http.ResponseWriter, r *http.Request) {
		resource := r.PathValue("resource")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("deleted=" + resource))
	})

	// Start the App and wait for it to be responsive
	startServer(app)

	// Test that method-specific handler is called for DELETE
	t.Run("method-specific handler takes precedence", func(t *testing.T) {
		url := fmt.Sprintf("http://%s/resource123", testAddress)
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			t.Errorf("failed to create request: %v", err)
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("get request failed: %v", err)
		}

		defer func() {
			err := res.Body.Close()
			if err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("failed to read response body: %v", err)
		}

		assert.Equal(t, http.StatusOK, res.StatusCode, "status code should be OK")
		assert.Equal(
			t,
			"deleted=resource123",
			string(body),
			"method-specific handler should be called",
		)
	})

	// Test that generic handler is called for GET
	t.Run("generic handler used when no method-specific", func(t *testing.T) {
		url := fmt.Sprintf("http://%s/resource456", testAddress)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Errorf("failed to create request: %v", err)
		}

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("get request failed: %v", err)
		}

		defer func() {
			err := res.Body.Close()
			if err != nil {
				t.Errorf("failed to close response body: %v", err)
			}
		}()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("failed to read response body: %v", err)
		}

		assert.Equal(t, http.StatusOK, res.StatusCode, "status code should be OK")
		assert.Equal(t, "resource=resource456", string(body), "generic handler should be called")
	})
}
