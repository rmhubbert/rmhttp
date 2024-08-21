package rmhttp

import "time"

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

func (tos *timeoutService) ApplyTimeout(handler Handler) Handler {
	return handler
}
