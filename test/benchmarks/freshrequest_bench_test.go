package benchmarks

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/rmhubbert/rmhttp/v5"
)

// ------------------------------------------------------------------------------------------------
// FRESH-REQUEST ROUTING
//
// Since Go 1.22 the net/http server creates a brand-new *http.Request for every request, even on
// a keep-alive connection, and the mux fills its path-value state per dispatch. Reused-request
// loops therefore understate the cost of anything that writes request path values (the first
// SetPathValue on a request allocates a backing map), so this file measures what a server
// actually pays: one genuinely fresh request per iteration, measured against a bare ServeMux
// running the identical pattern and handler.
// ------------------------------------------------------------------------------------------------

// newFreshRequest builds a production-shaped GET request the way http.ReadRequest does, so
// URL.EscapedPath round-trips escaped input.
func newFreshRequest(escapedPath string) *http.Request {
	u, err := url.ParseRequestURI(escapedPath)
	if err != nil {
		panic(err)
	}
	return &http.Request{
		Method:     http.MethodGet,
		URL:        u,
		RequestURI: escapedPath,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Host:       "example.com",
	}
}

func Benchmark_FreshRequest_Routing(b *testing.B) {
	shapes := []struct {
		name    string
		pattern string
		path    string
		params  []string
	}{
		{"exact", "/exact", "/exact", nil},
		{"1param", "/users/{id}", "/users/123", []string{"id"}},
		{"2param", "/a/{x}/b/{y}", "/a/1/b/2", []string{"x", "y"}},
		{"3param", "/a/{x}/b/{y}/c/{z}", "/a/1/b/2/c/3", []string{"x", "y", "z"}},
		{"tail", "/files/{path...}", "/files/path/to/file.txt", []string{"path"}},
		{"escaped", "/users/{id}", "/users/a%20b%2Fd", []string{"id"}},
	}

	for _, shape := range shapes {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, name := range shape.params {
				_ = r.PathValue(name)
			}
			_, _ = w.Write([]byte("ok"))
		})

		stdMux := http.NewServeMux()
		stdMux.Handle(http.MethodGet+" "+shape.pattern, handler)

		// Default App: no error handlers registered, so the Router passes every request
		// straight to the mux.
		app := rmhttp.New()
		app.Handle(http.MethodGet, shape.pattern, handler)
		app.Compile()

		// Intercepted App: a registered error handler (and its probe) on every request.
		appIntercepted := rmhttp.New()
		appIntercepted.Handle(http.MethodGet, shape.pattern, handler)
		appIntercepted.StatusNotFoundHandler(legacyErrorHandler(http.StatusNotFound))
		appIntercepted.StatusMethodNotAllowedHandler(
			legacyErrorHandler(http.StatusMethodNotAllowed),
		)
		appIntercepted.Compile()

		b.Run(shape.name+"/stdlib", benchFresh(stdMux, shape.path))
		b.Run(shape.name+"/rmhttp", benchFresh(app.Router, shape.path))
		b.Run(shape.name+"/rmhttp-intercepted", benchFresh(appIntercepted.Router, shape.path))
	}
}

// legacyErrorHandler answers with a status and its precomputed status-text body, like the
// library's former built-in default error handlers, so the intercepted columns stay comparable
// with the measurements taken while those defaults existed.
func legacyErrorHandler(code int) http.HandlerFunc {
	body := []byte(http.StatusText(code))
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
		_, _ = w.Write(body)
	}
}

func benchFresh(h http.Handler, path string) func(b *testing.B) {
	return func(b *testing.B) {
		w := newReuseWriter()
		defer w.release()

		b.ReportAllocs()
		for b.Loop() {
			w.reset()
			h.ServeHTTP(w, newFreshRequest(path))
		}
	}
}

// Benchmark_FreshRequest_404 measures the unmatched-request path, which the interception machinery
// exists to make configurable.
func Benchmark_FreshRequest_404(b *testing.B) {
	register := func(intercept bool) http.Handler {
		app := rmhttp.New()
		app.Get("/exists", func(w http.ResponseWriter, _ *http.Request) {})
		if intercept {
			app.StatusNotFoundHandler(legacyErrorHandler(http.StatusNotFound))
		}
		app.Compile()
		return app.Router
	}
	variants := []struct {
		name string
		h    http.Handler
	}{
		{"default", register(false)},
		{"intercepted", register(true)},
	}

	for _, variant := range variants {
		b.Run(variant.name, benchFresh(variant.h, "/missing"))
	}
}
