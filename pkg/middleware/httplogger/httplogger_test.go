package httplogger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// HTTP LOGGER TESTS
// ------------------------------------------------------------------------------------------------

var out = &bytes.Buffer{}

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(out, nil)))
	exitCode := m.Run()
	os.Exit(exitCode)
}

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

	for _, test := range tests {
		out.Reset()
		t.Run(test.name, func(t *testing.T) {
			handler := Middleware()(test.handler)
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

// logEntry mirrors the JSON fields the middleware emits that tests inspect.
type logEntry struct {
	Level  string `json:"level"`
	Status int    `json:"status"`
	Size   int    `json:"size"`
	IP     string `json:"ip"`
}

// serveAndDecodeLog runs one request through the middleware and decodes the
// single JSON log entry it must emit. It is only called from sequential tests
// because the package-level slog buffer shared with TestMain is not
// concurrency-safe.
func serveAndDecodeLog(
	t *testing.T,
	mw func(http.Handler) http.Handler,
	req *http.Request,
	status int,
) logEntry {
	t.Helper()

	out.Reset()
	handler := mw(http.HandlerFunc(createTestHandlerFunc(status, "body")))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	entry := logEntry{}
	if err := json.Unmarshal(out.Bytes(), &entry); err != nil {
		t.Fatalf("expected exactly one valid JSON log entry, got %q: %v", out.String(), err)
	}
	return entry
}

// Test_clientIP pins the contract of the trust-all client IP resolver:
// leftmost usable X-Forwarded-For entry, then X-Real-IP, then the normalized
// peer address. Empty/"unknown"/unparseable entries are skipped, and every
// logged value is validated and normalized with net/netip.
func Test_clientIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		remoteAddr string
		xffLines   []string
		realIP     string
		expected   string
	}{
		// Peer address parsing (no proxy headers).
		{
			name:       "no headers returns peer",
			remoteAddr: "192.0.2.7:1234",
			expected:   "192.0.2.7",
		},
		{
			name:       "peer IPv6 with port strips brackets",
			remoteAddr: "[2001:db8::1]:8080",
			expected:   "2001:db8::1",
		},
		{
			name:       "peer IPv6 without port",
			remoteAddr: "2001:db8::1",
			expected:   "2001:db8::1",
		},
		{
			name:       "peer zone ID stripped",
			remoteAddr: "fe80::1%eth0",
			expected:   "fe80::1",
		},
		{
			name:       "IPv4-mapped peer normalized",
			remoteAddr: "[::ffff:192.0.2.7]:8080",
			expected:   "192.0.2.7",
		},
		{
			name:       "missing peer returns empty string",
			remoteAddr: "",
			expected:   "",
		},
		{
			name:       "invalid peer returns empty string",
			remoteAddr: "garbage:80",
			expected:   "",
		},
		{
			name:       "ambiguous unbracketed IPv6 with port yields empty string",
			remoteAddr: "::ffff:10.0.0.1:8080",
			expected:   "",
		},

		// X-Forwarded-For trust-all selection.
		{
			name:       "leftmost XFF entry wins",
			remoteAddr: "192.0.2.9:9999",
			xffLines:   []string{"198.51.100.66, 203.0.113.7, 10.0.0.2"},
			expected:   "198.51.100.66",
		},
		{
			name:       "single XFF returns the value",
			remoteAddr: "10.0.0.1:80",
			xffLines:   []string{"203.0.113.7"},
			expected:   "203.0.113.7",
		},
		{
			name:       "unknown entry skipped case-insensitively",
			remoteAddr: "10.0.0.1:80",
			xffLines:   []string{"Unknown, 203.0.113.7"},
			expected:   "203.0.113.7",
		},
		{
			name:       "empty entry skipped",
			remoteAddr: "10.0.0.1:80",
			xffLines:   []string{", , 203.0.113.7"},
			expected:   "203.0.113.7",
		},
		{
			name:       "unparseable entries skipped leftmost valid wins",
			remoteAddr: "10.0.0.1:80",
			xffLines:   []string{"garbage, 203.0.113.7"},
			expected:   "203.0.113.7",
		},
		{
			name:       "empty XFF value falls back to peer",
			remoteAddr: "10.0.0.1:80",
			xffLines:   []string{""},
			expected:   "10.0.0.1",
		},
		{
			name:       "duplicate XFF lines merged left to right",
			remoteAddr: "10.0.0.1:80",
			xffLines:   []string{"198.51.100.66", "203.0.113.7"},
			expected:   "198.51.100.66",
		},
		{
			name:       "XFF entry with port normalized",
			remoteAddr: "10.0.0.1:80",
			xffLines:   []string{"198.51.100.9:8080"},
			expected:   "198.51.100.9",
		},
		{
			name:       "bracketed IPv6 XFF entry normalized",
			remoteAddr: "10.0.0.1:80",
			xffLines:   []string{"[2001:db8::5]:8080"},
			expected:   "2001:db8::5",
		},
		{
			name:       "IPv4-mapped XFF entry unmapped",
			remoteAddr: "10.0.0.1:80",
			xffLines:   []string{"::ffff:203.0.113.7"},
			expected:   "203.0.113.7",
		},
		{
			name:       "XFF takes precedence over X-Real-IP",
			remoteAddr: "10.0.0.1:80",
			xffLines:   []string{"203.0.113.7"},
			realIP:     "198.51.100.5",
			expected:   "203.0.113.7",
		},
		{
			name:       "all-invalid XFF chain falls through to X-Real-IP",
			remoteAddr: "10.0.0.1:80",
			xffLines:   []string{"garbage"},
			realIP:     "203.0.113.9",
			expected:   "203.0.113.9",
		},

		// X-Real-IP handling.
		{
			name:       "X-Real-IP honored when no XFF present",
			remoteAddr: "10.0.0.1:80",
			realIP:     "203.0.113.9",
			expected:   "203.0.113.9",
		},
		{
			name:       "X-Real-IP with port normalized",
			remoteAddr: "10.0.0.1:80",
			realIP:     "203.0.113.9:1234",
			expected:   "203.0.113.9",
		},
		{
			name:       "unparseable X-Real-IP falls back to peer",
			remoteAddr: "192.0.2.3:9",
			realIP:     "not-an-ip",
			expected:   "192.0.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := newBenchRequest(tt.remoteAddr, tt.xffLines, tt.realIP)
			assert.Equal(t, tt.expected, clientIP(req))
		})
	}
}

// Test_Middleware verifies the client IP resolution end to end through the
// emitted log entry, for both the success and error log paths.
func Test_Middleware(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xffLines   []string
		realIP     string
		status     int
		expectedIP string
	}{
		{
			name:       "leftmost XFF entry is logged",
			remoteAddr: "192.0.2.9:9999",
			xffLines:   []string{"198.51.100.66, 203.0.113.7"},
			status:     http.StatusOK,
			expectedIP: "198.51.100.66",
		},
		{
			name:       "single XFF entry returns the value",
			remoteAddr: "192.0.2.9:9999",
			xffLines:   []string{"203.0.113.7"},
			status:     http.StatusOK,
			expectedIP: "203.0.113.7",
		},
		{
			name:       "X-Real-IP fallback honored",
			remoteAddr: "192.0.2.9:9999",
			realIP:     "203.0.113.9",
			status:     http.StatusOK,
			expectedIP: "203.0.113.9",
		},
		{
			name:       "bracketed IPv6 peer normalized",
			remoteAddr: "[2001:db8::1]:8080",
			status:     http.StatusOK,
			expectedIP: "2001:db8::1",
		},
		{
			name:       "peer used when no headers present",
			remoteAddr: "192.0.2.7:1234",
			status:     http.StatusOK,
			expectedIP: "192.0.2.7",
		},
		{
			name:       "error path resolves IP the same way",
			remoteAddr: "192.0.2.9:9999",
			xffLines:   []string{"9.9.9.9"},
			status:     http.StatusInternalServerError,
			expectedIP: "9.9.9.9",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := newBenchRequest(test.remoteAddr, test.xffLines, test.realIP)
			entry := serveAndDecodeLog(t, Middleware(), req, test.status)
			assert.Equal(t, test.expectedIP, entry.IP)
			if test.status >= http.StatusBadRequest {
				assert.Equal(t, "ERROR", entry.Level)
			} else {
				assert.Equal(t, "INFO", entry.Level)
			}
		})
	}
}

// Test_Middleware_AlwaysLogs pins the guarantee that exactly one log entry is
// emitted per request even when the client IP cannot be resolved; ip:"" is the
// only acceptable degradation.
func Test_Middleware_AlwaysLogs(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xffLines   []string
		status     int
	}{
		{
			name:       "success path with unresolvable peer",
			remoteAddr: "",
			status:     http.StatusOK,
		},
		{
			name:       "success path with invalid peer and garbage XFF",
			remoteAddr: "not-a-peer",
			xffLines:   []string{"also-not-an-ip"},
			status:     http.StatusOK,
		},
		{
			name:       "error path with unresolvable peer",
			remoteAddr: "",
			status:     http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out.Reset()
			req := newBenchRequest(test.remoteAddr, test.xffLines, "")
			handler := Middleware()(http.HandlerFunc(createTestHandlerFunc(test.status, "body")))
			handler.ServeHTTP(httptest.NewRecorder(), req)

			lines := bytes.Split(bytes.TrimRight(out.Bytes(), "\n"), []byte("\n"))
			assert.Len(t, lines, 1, "exactly one log entry must be emitted per request")

			entry := logEntry{}
			if err := json.Unmarshal(out.Bytes(), &entry); err != nil {
				t.Fatalf("expected exactly one valid JSON log entry, got %q: %v", out.String(), err)
			}
			assert.Empty(t, entry.IP, "ip must degrade to empty string, never skip the entry")
			if test.status >= http.StatusBadRequest {
				assert.Equal(t, "ERROR", entry.Level)
			} else {
				assert.Equal(t, "INFO", entry.Level)
			}
		})
	}
}
