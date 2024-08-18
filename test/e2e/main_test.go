package e2e

import (
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/rmhubbert/rmhttp"
)

var (
	defaultPort        = 8080
	testAddress string = "localhost:" + strconv.Itoa(defaultPort)

	handlerTests = []struct {
		name       string
		method     string
		pattern    string
		pathToTest string
		status     int
		body       string
		err        error
	}{
		{"GET the index", http.MethodGet, "/", "/", http.StatusOK, "get body", nil},
		{"POST to the index", http.MethodPost, "/", "/", http.StatusOK, "post body", nil},
		{"PUT to the index", http.MethodPut, "/", "/", http.StatusOK, "put body", nil},
		{"PATCH to the index", http.MethodPatch, "/", "/", http.StatusOK, "patch body", nil},
		{"DELETE to the index", http.MethodDelete, "/", "/", http.StatusOK, "delete body", nil},
		{"OPTIONS to the index", http.MethodOptions, "/", "/", http.StatusNoContent, "", nil},
		{"GET /test", http.MethodGet, "/test", "/test", http.StatusOK, "get body", nil},
		{"POST to /test", http.MethodPost, "/test", "/test", http.StatusOK, "post body", nil},
		{"PUT to /test", http.MethodPut, "/test", "/test", http.StatusOK, "put body", nil},
		{"PATCH to /test", http.MethodPatch, "/test", "/test", http.StatusOK, "patch body", nil},
		{"DELETE to /test", http.MethodDelete, "/test", "/test", http.StatusOK, "delete body", nil},
		{"OPTIONS to /test", http.MethodOptions, "/test", "/test", http.StatusNoContent, "", nil},
		{"GET /test/{id}", http.MethodGet, "/test/{id}", "/test/105", http.StatusOK, "get body", nil},
		{"POST to /test/{id}", http.MethodPost, "/test/{id}", "/test/105", http.StatusOK, "post body", nil},
		{"PUT to /test/{id}", http.MethodPut, "/test/{id}", "/test/105", http.StatusOK, "put body", nil},
		{"PATCH to /test/{id}", http.MethodPatch, "/test/{id}", "/test/105", http.StatusOK, "patch body", nil},
		{"DELETE to /test/{id}", http.MethodDelete, "/test/{id}", "/test/105", http.StatusOK, "delete body", nil},
		{"OPTIONS to /test/{id}", http.MethodOptions, "/test/{id}", "/test/105", http.StatusNoContent, "", nil},
	}
)

func createHandlerFunc(status int, body string, err error) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(status)
		w.Write([]byte(body))
		return err
	}
}

func createNetHTTPHandlerFunc(status int, body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		w.Write([]byte(body))
	}
}

// startServer starts the rmhttp.App in a go routine, and then calls waitForServer
// in order to ensure that the app is running, before returning.
func startServer(app *rmhttp.App) {
	go app.ListenAndServe()
	waitForServer(defaultPort)
}

// startServer starts the rmhttp.NetHTTPApp in a go routine, and then calls waitForServer
// in order to ensure that the app is running, before returning.
func startNetHTTPServer(app *rmhttp.NetHTTPApp) {
	go app.ListenAndServe()
	waitForServer(defaultPort)
}

// waitForServer attempts to establish a TCP connection to localhost:<port> in a
// reasonable amount of time. It returns upon a successful connection;
// otherwise exits with an error.
func waitForServer(p int) {
	port := strconv.Itoa(p)
	backoff := 50 * time.Millisecond
	for i := 0; i < 10; i++ {
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
