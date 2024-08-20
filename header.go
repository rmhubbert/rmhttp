package rmhttp

import "net/http"

// ------------------------------------------------------------------------------------------------
// HEADERABLE INTERFACE
// ------------------------------------------------------------------------------------------------
// The Headerable interface allows any type that implements the interface to have HTTP headers
// associated with it within rmhttp.
type Headerable interface {
	Headers() map[string]string
}

func newHeaderService(logger Logger) *headerService {
	return &headerService{
		logger: logger,
	}
}

type headerService struct {
	logger Logger
}

func (mws *headerService) ApplyHeaders(w http.ResponseWriter, h Headerable) {
	for key, value := range h.Headers() {
		w.Header().Add(key, value)
	}
}
