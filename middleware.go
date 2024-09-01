package rmhttp

// ------------------------------------------------------------------------------------------------
// MIDDLEWARE
// ------------------------------------------------------------------------------------------------

type MiddlewareFunc func(Handler) Handler

// applyMiddleware wraps the passed Handler with each of the middleware functions passed.
func applyMiddleware(next Handler, middlewares []MiddlewareFunc) Handler {
	if len(middlewares) == 0 {
		return next
	}
	// loop backwards to maintain middlewares order
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}
	return next
}
