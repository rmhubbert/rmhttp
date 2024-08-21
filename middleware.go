package rmhttp

func newMiddlewareService(logger Logger) *middlewareService {
	return &middlewareService{
		logger: logger,
	}
}

type middlewareService struct {
	logger Logger
}

func (mws *middlewareService) ApplyMiddleware(handler Handler) Handler {
	return handler
}
