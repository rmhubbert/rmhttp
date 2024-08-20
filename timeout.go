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

func newTimeoutService(config TimeoutConfig, logger Logger) *timeoutService {
	return &timeoutService{
		config: config,
		logger: logger,
	}
}

type timeoutService struct {
	config TimeoutConfig
	logger Logger
}

func (tos *timeoutService) ApplyTimeout(t Timeoutable) Timeoutable {
	return t
}
