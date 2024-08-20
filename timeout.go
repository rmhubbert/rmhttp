package rmhttp

import "time"

// ------------------------------------------------------------------------------------------------
// TIMEOUTABLE INTERFACE
// ------------------------------------------------------------------------------------------------
// The Timeoutable interface allows any type that implements the interface to have a timeout
// associated with it within rmhttp.
type Timeoutable interface {
	Timeout() Timeout
	Handler() Handler
}

type Timeout time.Duration

type timeoutService struct{}

func (tos *timeoutService) ApplyTimeout(t Timeoutable) Timeoutable {
	return t
}
