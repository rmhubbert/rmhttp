package rmhttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestLogEntry struct {
	Level  string `json:"level"`
	Status int    `json:"status"`
}

func Test_HTTPErrorLoggerMiddleware(t *testing.T) {
	testPattern := "/test"
	testBody := "test body"
	// ErrSentinel := errors.New("sentinel error")

	tests := []struct {
		name          string
		expectedCode  int
		errorExpected bool
		route         *Route
	}{
		{
			"an error is logged when response status is 400 and no error is returned from handler",
			http.StatusBadRequest,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(createTestHandlerFunc(http.StatusBadRequest, testBody, nil)),
			),
		},
		{
			"an error is logged when response status is 500 and no error is returned from handler",
			http.StatusInternalServerError,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(createTestHandlerFunc(http.StatusInternalServerError, testBody, nil)),
			),
		},
		{
			"an error is not logged when response status is 200 and no error is returned from handler",
			http.StatusOK,
			false,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(createTestHandlerFunc(http.StatusOK, testBody, nil)),
			),
		},
		{
			"an error is logged when response status is 400 and an error is returned from handler",
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
			"an error is logged when response status is 200 and an error is returned from handler",
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
			"an error is logged when response status is 200 and an HTTPError is returned from handler",
			http.StatusBadRequest,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(
					createTestHandlerFunc(
						http.StatusOK,
						testBody,
						NewHTTPError(errors.New("error!"), http.StatusBadRequest),
					),
				),
			),
		},
		{
			"an error is logged when response status is 500 and an HTTPError is returned from handler",
			http.StatusBadRequest,
			true,
			NewRoute(
				http.MethodGet,
				testPattern,
				HandlerFunc(
					createTestHandlerFunc(
						http.StatusInternalServerError,
						testBody,
						NewHTTPError(errors.New("error!"), http.StatusBadRequest),
					),
				),
			),
		},
	}

	out := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(out, nil))

	for _, test := range tests {
		out.Reset()
		t.Run(test.name, func(t *testing.T) {
			test.route.Handler = applyMiddleware(
				test.route.Handler,
				[]MiddlewareFunc{HTTPErrorLoggerMiddleware(logger)},
			)

			url := fmt.Sprintf("http://%s%s", testAddress, testPattern)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			_ = test.route.Handler.ServeHTTPWithError(w, req)

			// fmt.Println("log: ", out.String(), " :END")
			log := TestLogEntry{}
			if err = json.Unmarshal(out.Bytes(), &log); err != nil {
				t.Errorf("cannot unmarshal log entry JSON: %v", err.Error())
			}

			if test.errorExpected {
				assert.Equal(t, test.expectedCode, log.Status, "they should be equal")
				assert.Equal(t, "ERROR", log.Level, "they should be equal")
			} else {
				assert.Equal(t, "INFO", log.Level, "they should be equal")
			}
		})
	}
}
