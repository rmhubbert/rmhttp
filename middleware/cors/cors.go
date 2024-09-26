package cors

import (
	"errors"
	"net/http"

	"github.com/rmhubbert/rmhttp"
	rsc "github.com/rs/cors"
)

type Options = rsc.Options

func Middleware(options ...Options) rmhttp.MiddlewareFunc {
	optionsPassthrough := false
	successStatus := http.StatusNoContent

	opt := Options{
		OptionsSuccessStatus: successStatus,
		OptionsPassthrough:   optionsPassthrough,
	}
	if len(options) > 0 {
		opt = options[0]
	}
	c := rsc.New(opt)

	return func(next rmhttp.Handler) rmhttp.Handler {
		return rmhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
			cw := rmhttp.NewCaptureWriter(w)
			c.HandlerFunc(cw, r)

			header := w.Header()
			for key, values := range cw.Header() {
				for _, value := range values {
					header.Add(key, value)
				}
			}

			if r.Method != http.MethodOptions {
				return next.ServeHTTPWithError(w, r)
			}

			if r.Header.Get("Access-Control-Request-Method") == "" {
				w.WriteHeader(http.StatusBadRequest)
				return rmhttp.NewHTTPError(
					errors.New("you must specify a Access-Control-Request-Method"),
					http.StatusBadRequest,
				)
			}

			if optionsPassthrough {
				return next.ServeHTTPWithError(w, r)
			} else {
				cw.Persist()
				return nil
			}
		})
	}
}
