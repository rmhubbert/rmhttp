package rmhttp

import "net/http"

// ------------------------------------------------------------------------------------------------
// MIDDLEWARE
// ------------------------------------------------------------------------------------------------

// applyMiddleware wraps the passed Handler with each of the middleware functions passed.
// Middleware is applied in reverse order, so that the first middleware in the slice
// wraps all subsequent middleware and the final handler.
//
// Example: applyMiddleware(handler, [A, B, C]) produces: A(B(C(handler)))
// Request flow: A → B → C → handler
// Response flow: handler → C → B → A
//
// This maintains the intuitive order where middleware added first runs first on requests
// and last on responses (decorator pattern).
func applyMiddleware(
	next http.Handler,
	middlewares []func(http.Handler) http.Handler,
) http.Handler {
	if len(middlewares) == 0 {
		return next
	}
	// loop backwards to maintain middlewares order
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}
	return next
}
