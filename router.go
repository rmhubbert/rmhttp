package rmhttp

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
)

// ------------------------------------------------------------------------------------------------
// ROUTER
// ------------------------------------------------------------------------------------------------

// The Router loads Routes into the underlying HTTP request multiplexer, as well as handling each
// request, ensuring that ResponseWriter and Request objects are properly configured. The Router
// also manages custom error handlers to ensure that the HTTP Error Handler can operate
// properly.
type Router struct {
	Mux *http.ServeMux

	// errorMu serializes AddErrorHandler writes; serving reads the immutable snapshot held in
	// errorHandlers, so the hot lookup is a single atomic load plus a map index — no locking,
	// no type assertion. Writes clone-on-store: registration is complete before the server
	// starts in normal use, and stays safe for the concurrent edge.
	errorMu       sync.Mutex
	errorHandlers atomic.Pointer[map[int]http.Handler]

	// routes maps the exact pattern strings registered through Handle (e.g. "GET /users/{id}")
	// to the wrapper handler registered on the Mux plus the path parameters parsed from the
	// pattern. A pattern's presence and handler identity is what enables the single-match fast
	// path in ServeHTTP; anything else the mux may return (stdlib-generated redirect handlers,
	// patterns registered directly on the Mux) keeps the full stdlib path. The map is written
	// only during registration — which completes before the server starts, as with Mux.Handle
	// itself — and read-only while serving.
	routes map[string]registeredRoute
}

// registeredRoute records what Router.Handle registered for one pattern.
type registeredRoute struct {
	handler http.Handler
	params  []routeParam
}

// routedHandler marks the handlers Router.Handle registers on the Mux so ServeHTTP can recognize
// them. The mark is a pointer so ServeHTTP's identity comparison can never hit the runtime panic
// that comparing two http.Handler values with non-comparable dynamic types (http.HandlerFunc, for
// example) would cause.
type routedHandler struct {
	http.Handler
}

// routeParam describes one wildcard of a registered pattern: the parameter name, the index of the
// pattern segment it occupies when the path is split on "/", and whether it is a trailing
// "{name...}" wildcard that captures the remaining segments.
type routeParam struct {
	name     string
	segIndex int
	tail     bool
}

// NewRouter intialises, creates, and then returns a pointer to a Router.
func NewRouter() *Router {
	rt := &Router{
		Mux:    http.NewServeMux(),
		routes: make(map[string]registeredRoute),
	}
	empty := map[int]http.Handler{}
	rt.errorHandlers.Store(&empty)
	return rt
}

// ServeHTTP allows the Router to fulfill the http.Handler interface, meaning that we can use it as
// a handler for the underlying HTTP request multiplexer (which by default is a http.ServeMux).
//
// We also intercept any error handlers returned by the underlying mux, and replace them with any
// custom error handlers that have been registered.
//
// Matching is delegated to the Mux exactly once per request. The mux's Handler method does not
// populate named path wildcards (only its ServeHTTP does), so for patterns registered through
// Handle we re-apply the wildcards the mux matched — using pattern metadata parsed at
// registration time — and call the matched handler directly. Anything we did not register
// ourselves (stdlib-generated canonical/host-slash redirects and the like) keeps the full
// stdlib ServeHTTP behavior, path values included.
//
// Note: the per-request errorCapture instance used on the error path is pooled and must not be
// shared across concurrent requests.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// The mux answers the OPTIONS "*" request form with 400 inside ServeHTTP, before any
	// pattern matching. Hand that special case straight to the stdlib.
	if r.RequestURI == "*" {
		rt.Mux.ServeHTTP(w, r)
		return
	}

	handler, pattern := rt.Mux.Handler(r)

	// When ServeMux.Handler() returns an empty pattern, it means either:
	//
	// 1. No route matched (404)
	// 2. Method not allowed for matched pattern (405)
	//
	// In both cases, we execute the mux's error handler exactly once into a pooled errorCapture
	// and decide what to send: a custom error handler if registered, otherwise a replay of the
	// captured status and body. The handler is never executed a second time.
	//
	// Header() inside the capture delegates to w, so the handler's own headers (Allow on a 405,
	// Content-Type/nosniff on both) land on the real writer exactly as they would without the
	// capture — that is the observable behavior custom error handlers already rely on.
	if pattern == "" && handler != nil {
		ec := newErrorCapture(w)
		handler.ServeHTTP(ec, r)

		// Only consider a custom handler when the captured status is an error (not the default
		// 200 that a non-writing handler would leave behind).
		var customHandler http.Handler
		if ec.status != 0 && ec.status != http.StatusOK {
			customHandler = (*rt.errorHandlers.Load())[ec.status]
		}

		if customHandler != nil {
			// Return the capture to the pool before the custom handler writes to the real
			// writer, so the pooled body is never live while the real response is being built.
			ec.release()
			customHandler.ServeHTTP(w, r)
			return
		}

		// No custom handler: replay the single captured response instead of re-running the
		// mux's handler against the real writer.
		w.WriteHeader(ec.status)
		if ec.body.Len() > 0 {
			_, _ = w.Write(ec.body.Bytes())
		}
		ec.release()
		return
	}

	// Single-pass fast path: the mux returned the very handler rmhttp registered for this
	// pattern, so there is no stdlib-generated behavior to replicate. Apply the path values
	// the mux matched but will not expose, then run the handler directly.
	if route, ok := rt.routes[pattern]; ok && handler == route.handler {
		if len(route.params) > 0 {
			// Matching used the escaped path (cleaned); the identity check above guarantees the
			// mux matched it without a redirect, so a leading slash is expected. CONNECT requests
			// are not canonicalized, so anything else falls back to the full stdlib path.
			if escapedPath := r.URL.EscapedPath(); len(escapedPath) > 0 && escapedPath[0] == '/' {
				for _, p := range route.params {
					if value := extractParam(escapedPath, p); value != "" {
						r.SetPathValue(p.name, value)
					}
				}
			}
		}
		r.Pattern = pattern
		handler.ServeHTTP(w, r)
		return
	}

	// Stdlib-generated handler (canonical-path or host-slash redirect, etc.) or a pattern that
	// was registered directly on the Mux: let the mux reproduce its exact behavior, path values
	// included.
	rt.Mux.ServeHTTP(w, r)
}

// AddErrorHandler maps the passed response code and handler. These error handlers will be used
// instead of the http.Handler equivalents when available. The first handler registered for a
// given code wins; later registrations for the same code are ignored.
func (rt *Router) AddErrorHandler(code int, handler http.Handler) {
	rt.errorMu.Lock()
	defer rt.errorMu.Unlock()

	old := *rt.errorHandlers.Load()
	if _, exists := old[code]; exists {
		return
	}
	next := make(map[int]http.Handler, len(old)+1)
	maps.Copy(next, old)
	next[code] = handler
	rt.errorHandlers.Store(&next)
}

// HasErrorHandlers returns true if the Router has any error handlers registered.
func (rt *Router) HasErrorHandlers() bool {
	return len(*rt.errorHandlers.Load()) > 0
}

// Handle registers the passed Route with the underlying HTTP request multiplexer.
func (rt *Router) Handle(method string, pattern string, handler http.Handler) {
	key := fmt.Sprintf("%s %s", method, pattern)
	wrapped := &routedHandler{Handler: handler}
	rt.Mux.Handle(key, wrapped)

	route := registeredRoute{handler: wrapped}
	if params, ok := parsePathParams(pattern); ok {
		route.params = params
	}
	rt.routes[key] = route
}

// parsePathParams extracts the path parameters from a registered path pattern. Patterns that the
// single-match fast path cannot reproduce exactly — the empty pattern, patterns with a trailing
// slash (which the mux compiles to an anonymous wildcard), anonymous "{...}" wildcards and other
// unusual segment shapes — report ok=false so their requests keep the full stdlib ServeHTTP path.
// Registration-time only, so the allocation from splitting is irrelevant.
func parsePathParams(pattern string) ([]routeParam, bool) {
	if pattern == "" || pattern[0] != '/' || pattern[len(pattern)-1] == '/' {
		return nil, false
	}

	var (
		params  []routeParam
		lastIdx int
		tailIdx = -1
	)
	i := 0
	for seg := range strings.SplitSeq(pattern, "/") {
		lastIdx = i
		if i > 0 && strings.HasPrefix(seg, "{") {
			if !strings.HasSuffix(seg, "}") {
				return nil, false
			}
			name, isTail := strings.CutSuffix(seg[1:len(seg)-1], "...")
			if name == "" || name == "$" || strings.Contains(name, "{") {
				return nil, false // anonymous, {$} or otherwise unusual wildcard
			}
			if isTail {
				tailIdx = i
			}
			params = append(params, routeParam{name: name, segIndex: i, tail: isTail})
		}
		i++
	}
	// A "{name...}" wildcard may only appear as the final segment.
	if tailIdx >= 0 && tailIdx != lastIdx {
		return nil, false
	}
	return params, true
}

// extractParam returns the value the ServeMux matched for one pattern wildcard. It re-walks the
// escaped request path the way the mux's routing tree matched it: segments split on "/", each
// captured value unescaped with url.PathUnescape (falling back to the raw text when invalid), and
// a tail wildcard capturing the remaining segments without their leading slash. The walk itself
// allocates nothing on paths without percent escapes.
func extractParam(escapedPath string, p routeParam) string {
	// Advance pos to the slash that precedes segment p.segIndex. Segments split from a path
	// beginning with "/" start at index 1, and wildcards never occupy segment 0.
	pos := 0
	for range p.segIndex - 1 {
		next := strings.IndexByte(escapedPath[pos+1:], '/')
		if next < 0 {
			return ""
		}
		pos += next + 1
	}

	if p.tail {
		return pathUnescape(escapedPath[pos+1:])
	}
	end := len(escapedPath)
	if next := strings.IndexByte(escapedPath[pos+1:], '/'); next >= 0 {
		end = pos + 1 + next
	}
	return pathUnescape(escapedPath[pos+1 : end])
}

// pathUnescape mirrors the stdlib mux behavior: an invalid escape sequence yields the raw text.
func pathUnescape(path string) string {
	unescaped, err := url.PathUnescape(path)
	if err != nil {
		return path
	}
	return unescaped
}
