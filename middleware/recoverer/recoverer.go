package recoverer

import (
	"net/http"

	"github.com/rmhubbert/rmhttp"
)

func Middleware() rmhttp.MiddlewareFunc {
	return func(next rmhttp.Handler) rmhttp.Handler {
		return rmhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			defer func() {
				if rvr := recover(); rvr != nil {
					if r.Header.Get("Connection") != "Upgrade" {
						w.WriteHeader(http.StatusInternalServerError)
					}
				}
			}()
			return next.ServeHTTPWithError(w, r)
		})
	}
}
