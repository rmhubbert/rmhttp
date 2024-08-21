package rmhttp

import "net/http"

func newHeaderService(logger Logger) *headerService {
	return &headerService{
		logger: logger,
	}
}

type headerService struct {
	logger Logger
}

func (mws *headerService) ApplyHeaders(w http.ResponseWriter, route *Route) {
	for key, value := range route.Headers() {
		w.Header().Add(key, value)
	}
}
