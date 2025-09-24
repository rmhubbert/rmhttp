package e2e

import (
	"bytes"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/rmhubbert/rmhttp/v5"
)

// ------------------------------------------------------------------------------------------------
// CONVENIENCE FUNCTIONS, CONSTANTS AND VARIABLES FOR E2E TESTING
// ------------------------------------------------------------------------------------------------

const defaultPort int = 8321

var out = &bytes.Buffer{}
var testAddress string = "localhost:" + strconv.Itoa(defaultPort)
var handlerTests = []struct {
	name       string
	method     string
	pattern    string
	pathToTest string
	status     int
	body       string
}{
	{"GET the index", http.MethodGet, "/", "/", http.StatusOK, "get body"},
	{"POST to the index", http.MethodPost, "/", "/", http.StatusOK, "post body"},
	{"PUT to the index", http.MethodPut, "/", "/", http.StatusOK, "put body"},
	{"PATCH to the index", http.MethodPatch, "/", "/", http.StatusOK, "patch body"},
	{"DELETE to the index", http.MethodDelete, "/", "/", http.StatusOK, "delete body"},
	{"OPTIONS to the index", http.MethodOptions, "/", "/", http.StatusNoContent, ""},
	{"GET /test", http.MethodGet, "/test", "/test", http.StatusOK, "get body"},
	{"POST to /test", http.MethodPost, "/test", "/test", http.StatusOK, "post body"},
	{"PUT to /test", http.MethodPut, "/test", "/test", http.StatusOK, "put body"},
	{"PATCH to /test", http.MethodPatch, "/test", "/test", http.StatusOK, "patch body"},
	{"DELETE to /test", http.MethodDelete, "/test", "/test", http.StatusOK, "delete body"},
	{"OPTIONS to /test", http.MethodOptions, "/test", "/test", http.StatusNoContent, ""},
	{
		"GET /test/{id}",
		http.MethodGet,
		"/test/{id}",
		"/test/105",
		http.StatusOK,
		"get body",
	},
	{
		"POST to /test/{id}",
		http.MethodPost,
		"/test/{id}",
		"/test/105",
		http.StatusOK,
		"post body",
	},
	{
		"PUT to /test/{id}",
		http.MethodPut,
		"/test/{id}",
		"/test/105",
		http.StatusOK,
		"put body",
	},
	{
		"PATCH to /test/{id}",
		http.MethodPatch,
		"/test/{id}",
		"/test/105",
		http.StatusOK,
		"patch body",
	},
	{
		"DELETE to /test/{id}",
		http.MethodDelete,
		"/test/{id}",
		"/test/105",
		http.StatusOK,
		"delete body",
	},
	{
		"OPTIONS to /test/{id}",
		http.MethodOptions,
		"/test/{id}",
		"/test/105",
		http.StatusNoContent,
		"",
	},
}

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(out, nil)))
	exitCode := m.Run()
	os.Exit(exitCode)
}

// createTestHandlerFunc creates, initialises, and returns a rmhttp.HandlerFunc compatible function.
func createTestHandlerFunc(
	status int,
	body string,
) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}
}

// startServer starts the rmhttp.App in a go routine, and then calls waitForServer in order to
// ensure that the app is running, before returning.
func startServer(app *rmhttp.App) {
	go func() {
		_ = app.ListenAndServe()
	}()
	waitForServer(defaultPort)
}

// waitForServer attempts to establish a TCP connection to localhost:<port> in a reasonable amount
// of time. It returns upon a successful connection; otherwise it will exit with an error.
func waitForServer(p int) {
	port := strconv.Itoa(p)
	backoff := 50 * time.Millisecond
	for range 10 {
		conn, err := net.DialTimeout("tcp", ":"+port, 1*time.Second)
		if err != nil {
			time.Sleep(backoff)
			continue
		}
		err = conn.Close()
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	log.Fatalf("server on port %s is not up after 10 attempts", port)
}
