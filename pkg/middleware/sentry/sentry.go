package sentry

import (
	"net/http"

	sentryhttp "github.com/getsentry/sentry-go/http"
)

type Options = sentryhttp.Options

func Middleware(options Options) func(http.Handler) http.Handler {
	s := sentryhttp.New(options)
	return s.Handle
}
