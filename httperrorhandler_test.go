package rmhttp

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type CustomError struct{}

func (ce CustomError) Error() string {
	return "custom error"
}

func Test_HTTPErrorHandlerMiddleware(t *testing.T) {
	testPattern := "/test"
	testBody := "test body"
	ErrSentinel := errors.New("sentinel error")

	tests := []struct {
		name          string
		expectedCode  int
		errorExpected bool
		route         *Route
	}{
		{
			"an HTTPError is created when response status is 400 and no error is returned from handler",
			http.StatusBadRequest,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(createTestHandlerFunc(http.StatusBadRequest, testBody, nil)),
			),
		},
		{
			"an HTTPError is created when response status is 500 and no error is returned from handler",
			http.StatusInternalServerError,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(createTestHandlerFunc(http.StatusInternalServerError, testBody, nil)),
			),
		},
		{
			"an HTTPError is not created when response status is 200 and no error is returned from handler",
			http.StatusOK,
			false,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(createTestHandlerFunc(http.StatusOK, testBody, nil)),
			),
		},
		{
			"an HTTPError is created when response status is 400 and an error is returned from handler",
			http.StatusBadRequest,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(
					createTestHandlerFunc(http.StatusBadRequest, testBody, errors.New("error!")),
				),
			),
		},
		{
			"an HTTPError is created when response status is 200 and an error is returned from handler",
			http.StatusInternalServerError,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(
					createTestHandlerFunc(http.StatusOK, testBody, errors.New("error!")),
				),
			),
		},
		{
			"the HTTPError has priority when response status is 200 and an HTTPError is returned from handler",
			http.StatusBadRequest,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(
					createTestHandlerFunc(http.StatusOK,
						testBody,
						NewHTTPError(errors.New("error!"), http.StatusBadRequest),
					),
				),
			),
		},
		{
			"the HTTPError has priority when response status is 400 and an HTTPError is returned from handler",
			http.StatusForbidden,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(
					createTestHandlerFunc(http.StatusBadRequest,
						testBody,
						NewHTTPError(errors.New("error!"), http.StatusForbidden),
					),
				),
			),
		},
		{
			"an HTTPError is created with the correct status code when a registered sentinel error is returned from handler",
			http.StatusBadRequest,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(
					createTestHandlerFunc(http.StatusOK,
						testBody,
						ErrSentinel,
					),
				),
			),
		},
		{
			"an HTTPError is created with the correct status code when a registered custom error is returned from handler",
			http.StatusForbidden,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(
					createTestHandlerFunc(http.StatusOK,
						testBody,
						CustomError{},
					),
				),
			),
		},
		{
			"an HTTPError is created with the correct status code when a registered sentinel error is wrapped and returned from handler",
			http.StatusBadRequest,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(
					createTestHandlerFunc(http.StatusOK,
						testBody,
						fmt.Errorf("wrapped err: %w", ErrSentinel),
					),
				),
			),
		},
		{
			"an HTTPError is created with the correct status code when a registered custom error is wrapped and returned from handler",
			http.StatusForbidden,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(
					createTestHandlerFunc(http.StatusOK,
						testBody,
						fmt.Errorf("wrapped err: %w", CustomError{}),
					),
				),
			),
		},
	}

	registeredErrors := map[error]int{
		ErrSentinel:   400,
		CustomError{}: 403,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.route.Handler = applyMiddleware(
				test.route.Handler,
				[]MiddlewareFunc{HTTPErrorHandlerMiddleware(registeredErrors)},
			)

			url := fmt.Sprintf("http://%s%s", testAddress, testPattern)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			requestErr := test.route.Handler.ServeHTTPWithError(w, req)

			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, test.expectedCode, res.StatusCode, "they should be equal")

			if test.errorExpected {
				HTTPErr, ok := requestErr.(HTTPError)
				if !ok {
					t.Fatal("cannot convert error to HTTPError")
				}
				assert.IsType(t, HTTPError{}, requestErr, "they should be equal")
				assert.Equal(t, test.expectedCode, HTTPErr.Code, "they should be equal")
			} else {
				assert.NoError(t, requestErr, "it should be nil")
			}
		})
	}
}
