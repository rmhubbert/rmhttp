package cors

import (
	"net/http"

	rsc "github.com/rs/cors"
)

type Options = rsc.Options

func Middleware(options Options) func(http.Handler) http.Handler {
	c := rsc.New(options)
	return c.Handler
}
