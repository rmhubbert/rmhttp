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
// NET HTTP COMPATIBLE APP E2E TESTS
// ------------------------------------------------------------------------------------------------
// Test_Handler tests binding an http.Handler to a method & pattern with a selection of route
// patterns, methods and status codes.
func Test_NetHTTPHandle(t *testing.T) {
	// Set up the App
	app := rmhttp.NewNetHTTP()
	defer func() {
		_ = app.Shutdown()
	}()

	// Add handlers for all of our tests
	for _, test := range handlerTests {
		route := app.Handle(
			test.method,
			test.pattern,
			http.HandlerFunc(createTestNetHTTPHandlerFunc(test.status, test.body)),
		)
		assert.Equal(
			t,
			fmt.Sprintf("%s %s", strings.ToUpper(test.method), strings.ToLower(test.pattern)),
			route.String(),
		)
	}

	// Start the NetHTTPApp and wait for it to be responsive
	startNetHTTPServer(app)

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

// Test_HandlerFunc tests binding an http.Handler to a method & pattern with a selection of route
// patterns, methods and status codes.
func Test_NetHTTPHandleFunc(t *testing.T) {
	// Set up the App
	app := rmhttp.NewNetHTTP()
	defer func() {
		_ = app.Shutdown()
	}()

	// Add handlers for all of our tests
	for _, test := range handlerTests {
		route := app.HandleFunc(
			test.method,
			test.pattern,
			createTestNetHTTPHandlerFunc(test.status, test.body),
		)
		assert.Equal(
			t,
			fmt.Sprintf("%s %s", strings.ToUpper(test.method), strings.ToLower(test.pattern)),
			route.String(),
		)
	}

	// Start the NetHTTPApp and wait for it to be responsive
	startNetHTTPServer(app)

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
