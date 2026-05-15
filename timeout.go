package rmhttp

import (
	"net/http"
	"time"
)

// ------------------------------------------------------------------------------------------------
// TIMEOUT
// ------------------------------------------------------------------------------------------------

// Timeout encapsulates a duration and message that should be used for applying timeouts to
// Route handlers, with a specific error message.
type Timeout struct {
	Duration time.Duration
	Message  string
	Enabled  bool
}

// NewTimeout creates, initialises and returns a pointer to a Timeout.
func NewTimeout(duration time.Duration, message string) Timeout {
	return Timeout{
		Duration: duration,
		Message:  message,
		Enabled:  true,
	}
}

// ------------------------------------------------------------------------------------------------
// TIMEOUT MIDDLEWARE
// ------------------------------------------------------------------------------------------------

// TimeoutMiddleware creates, initialises and returns a middleware function that will wrap the next
// handler in the stack with a timeout handler. The http.TimeoutHandler is created once when the
// middleware is applied (at compile time), not per-request, avoiding per-request goroutine,
// channel, and context allocations.
func TimeoutMiddleware(timeout Timeout) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout.Duration, timeout.Message)
	}
}
