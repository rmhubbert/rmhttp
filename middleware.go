package rmhttp

// ------------------------------------------------------------------------------------------------
// MIDDLEWARE
// ------------------------------------------------------------------------------------------------

type MiddlewareFunc func(Handler) Handler

// ------------------------------------------------------------------------------------------------
// MIDDLEWARE SERVICE
// ------------------------------------------------------------------------------------------------

// middlewareService supplies functionality for manging global middleware and applying middleware
// to Handlers.
type middlewareService struct {
	logger Logger
	pre    []MiddlewareFunc
	post   []MiddlewareFunc
}

// newMiddlewareService creates, initialises, and then returns a pointer to a new middlewareService.
func newMiddlewareService(logger Logger) *middlewareService {
	return &middlewareService{
		logger: logger,
	}
}

// addPre stores a middleware function that will be globally applied to every handler passed to
// applyMiddleware. The middleware stored by addPre will be applied before any middlewares
// passed to applyMiddleware.
func (mws *middlewareService) addPre(middleware MiddlewareFunc) {
	mws.pre = append(mws.pre, middleware)
}

// addPost stores a middleware function that will be globally applied to every handler passed to
// applyMiddleware. The middleware stored by addPost will be applied after any middlewares
// passed to applyMiddleware.
func (mws *middlewareService) addPost(middleware MiddlewareFunc) {
	mws.post = append(mws.post, middleware)
}

// applyMiddleware wraps the passed Handler with each of the middleware functions passed. This
// function will also add any global pre & post middleware functions that have been set in
// the service before and after the passed middleware functions.
//
// As an example, if pre1 & pre2 have been previously set via middlewareService.addPre, post1
// & post have been set via middlewareService.addPost, and middlewares consists of mw1 &
// mw2, the flow would look like the follwoing after this function completes -
//
// pre1 -> pre2 -> mw1 -> mw2 -> post1 -> post2 -> handler -> post2 -> post1 -> mw2 -> mw1 -> pre2 -> pre1
func (mws *middlewareService) applyMiddleware(next Handler, middlewares []MiddlewareFunc) Handler {
	mw := append(mws.pre, middlewares...)
	mw = append(mw, mws.post...)

	if len(mw) == 0 {
		return next
	}
	// loop backwards to maintain middlewares order
	for i := len(mw) - 1; i >= 0; i-- {
		next = mw[i](next)
	}
	return next
}
