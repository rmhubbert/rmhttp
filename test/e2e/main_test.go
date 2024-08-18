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
)

func createHandlerFunc(status int, body string, err error) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(status)
		w.Write([]byte(body))
		return err
	}
}

// startServer starts the rmhttp.App in a go routine, anf then attempts to
// establish a TCP connection to localhost:<port> in a given amount of
// time. It returns upon a successful connection; otherwise exits
// with an error.
func startServer(app *rmhttp.App) {
	go app.Start()
	port := strconv.Itoa(defaultPort)

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
