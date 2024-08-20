package rmhttp

// ------------------------------------------------------------------------------------------------
// USABLE INTERFACE
// ------------------------------------------------------------------------------------------------
// The Usable interface allows any type that implements the interface to have middleware associated
// vwith it within rmhttp.
type Usable interface {
	Middleware() []func(Handler) Handler
	Handler() Handler
}

type middlewareService struct{}

func (mws *middlewareService) ApplyMiddleware(u Usable) Usable {
	return u
}
