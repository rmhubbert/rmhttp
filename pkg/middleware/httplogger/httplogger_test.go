package httplogger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// HTTP LOGGER TESTS
// ------------------------------------------------------------------------------------------------

type TestLogEntry struct {
	Level  string `json:"level"`
	Status int    `json:"status"`
}

func createTestHandlerFunc(
	status int,
	body string,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

func Test_HTTPLogger(t *testing.T) {
	testAddress := "localhost:8123"
	testPattern := "/test"
	testBody := "test body"

	tests := []struct {
		name          string
		expectedCode  int
		errorExpected bool
		handler       http.Handler
	}{
		{
			"an error is logged when response status is 400",
			http.StatusBadRequest,
			true,
			http.HandlerFunc(createTestHandlerFunc(http.StatusBadRequest, testBody)),
		},
		{
			"an error is logged when response status is 500",
			http.StatusInternalServerError,
			true,
			http.HandlerFunc(createTestHandlerFunc(http.StatusInternalServerError, testBody)),
		},
		{
			"an error is not logged when response status is 200",
			http.StatusOK,
			false,
			http.HandlerFunc(createTestHandlerFunc(http.StatusOK, testBody)),
		},
	}

	out := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(out, nil))

	for _, test := range tests {
		out.Reset()
		t.Run(test.name, func(t *testing.T) {
			handler := Middleware(logger)(test.handler)
			url := fmt.Sprintf("http://%s%s", testAddress, testPattern)
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Errorf("failed to create request: %v", err)
			}

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

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
