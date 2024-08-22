package rmhttp

func newMiddlewareService(logger Logger) *middlewareService {
	return &middlewareService{
		logger: logger,
	}
}

type middlewareService struct {
	logger Logger
	pre    []func(Handler) Handler
	post   []func(Handler) Handler
}

func (mws *middlewareService) addPre(middleware func(Handler) Handler) {
	mws.pre = append(mws.pre, middleware)
}

func (mws *middlewareService) addPost(middleware func(Handler) Handler) {
	mws.post = append(mws.post, middleware)
}

func (mws *middlewareService) ApplyMiddleware(handler Handler) Handler {
	return handler
}
