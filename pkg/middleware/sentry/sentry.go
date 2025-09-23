package sentry

import (
	"net/http"

	sen "github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
)

type Options = sentryhttp.Options
type ClientOptions = sen.ClientOptions

func Middleware(options Options) func(http.Handler) http.Handler {
	s := sentryhttp.New(options)
	return s.Handle
}

func Init(clientOptions ClientOptions) error {
	if err := sen.Init(clientOptions); err != nil {
		return err
	}
	return nil
}
