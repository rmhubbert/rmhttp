package rmhttp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// ------------------------------------------------------------------------------------------------
// CONCURRENT REGISTRATION
//
// The package doc promises that route registration is safe for concurrent use. The router's own
// lookup table must therefore never race with serving, mirroring how http.ServeMux itself guards
// its tree. These tests are the -race regression gate for that promise.
// ------------------------------------------------------------------------------------------------

func Test_Router_Handle_ConcurrentWithServeHTTP(t *testing.T) {
	const registrations = 500

	router := NewRouter()
	noop := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	router.Handle(http.MethodGet, "/warm", noop)

	var wg sync.WaitGroup

	// Register while serving. Before the atomic-snapshot fix, the map write in Handle raced with
	// the map read in ServeHTTP's fast path and the race detector flagged it.
	wg.Go(func() {
		for i := range registrations {
			router.Handle(http.MethodGet, fmt.Sprintf("/late%d", i), noop)
		}
	})

	for range 4 {
		// One request per goroutine: the mux writes r.Pattern/r.pat/r.matches on dispatch, so a
		// shared request would race in the stdlib itself, exactly as it does under a real server
		// where every request is a distinct object.
		wg.Go(func() {
			w := httptest.NewRecorder()
			for range 200 {
				router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/warm", nil))
			}
		})
	}

	wg.Wait()

	// Every concurrently registered route must be visible and dispatchable once registration
	// completes, proving the served goroutines raced a live, complete snapshot rather than a
	// torn map.
	for i := range registrations {
		path := fmt.Sprintf("/late%d", i)
		called := false
		router.Handle(http.MethodGet, path+"/done", http.HandlerFunc(
			func(http.ResponseWriter, *http.Request) { called = true },
		))
		router.ServeHTTP(
			httptest.NewRecorder(),
			httptest.NewRequest(http.MethodGet, path+"/done", nil),
		)
		assert.True(t, called, "route %s registered concurrently must dispatch", path)
	}
}

func Test_Router_AddErrorHandler_ConcurrentWithServeHTTP(t *testing.T) {
	router := NewRouter()
	router.Handle(http.MethodGet, "/exists", http.HandlerFunc(
		func(http.ResponseWriter, *http.Request) {},
	))

	var wg sync.WaitGroup
	wg.Go(func() {
		for i := range 200 {
			router.AddErrorHandler(
				http.StatusTeapot+i,
				http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
			)
		}
	})

	for range 4 {
		// Fresh request per iteration: see the note above on why requests are never shared.
		wg.Go(func() {
			w := httptest.NewRecorder()
			for range 200 {
				router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/missing", nil))
			}
		})
	}
	wg.Wait()
}
