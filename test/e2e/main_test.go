package e2e

import (
	"log"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/rmhubbert/rmhttp"
)

var (
	app         *rmhttp.App = rmhttp.New()
	testAddress string      = "localhost:8080"
)

func TestMain(m *testing.M) {
	go app.Start()
	waitForServer(strconv.Itoa(app.Server.Port))

	os.Exit(m.Run())
}

// waitForServer attempts to establish a TCP connection to localhost:<port>
// in a given amount of time. It returns upon a successful connection;
// otherwise exits with an error.
func waitForServer(port string) {
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
