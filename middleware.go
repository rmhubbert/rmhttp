package rmhttp

// ------------------------------------------------------------------------------------------------
// USABLE INTERFACE
// ------------------------------------------------------------------------------------------------
// The Usable interface allows any type that implements it to have middleware associated with it
// within rmhttp.
type Usable interface {
	Middleware() []func(Handler) Handler
	Handler() Handler
}
