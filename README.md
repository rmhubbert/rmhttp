# rmhttp

[![Go Reference](https://pkg.go.dev/badge/github.com/rmhubbert/rmhttp.svg)](https://pkg.go.dev/github.com/rmhubbert/rmhttp) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/rmhubbert/rmhttp?color=%23007D9C)
![GitHub Release Date](https://img.shields.io/github/release-date/rmhubbert/rmhttp?color=%23007D9C)
![GitHub commits since latest release](https://img.shields.io/github/commits-since/rmhubbert/rmhttp/latest?color=%23007D9C) [![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?color=%23007D9C)](CONTRIBUTING.md)

**rmhttp** provides a lightweight wrapper around the Go standard library HTTP server and router provided by [net/http](https://pkg.go.dev/net/http) that allows for the easy implementation of centralised error handling, groups, header management, and middleware (at the route, group and server level).

Handlers and middleware functions are kept as close to the standard library implementation as possible, with one addition; they can return an error. This allows you to simply return any errors from your handler, and have **rmhttp** transform the error into an HTTP response with a corresponding status code.

You can easily register any custom or sentinel error with **rmhttp** with the required HTTP status code, and the library will handle creating the correct response for you. It's also worth noting that you can also set your responses manually, if you don't want to use that particular functionality.

In addition to the centralised error handling, **rmhttp** also offers convenience methods for binding handlers to all of the available HTTP methods, easy grouping (and subgrouping) of your routes, plus header and middleware management at the global, group and route level.

[Go v1.23](https://go.dev/doc/go1.23) is the minimum supported version, as **rmhttp** takes advantage of the [net/http routing enhancements](https://go.dev/blog/routing-enhancements) released in v1.22.

## Installation

Run the following command from your project root directory to install **rmhttp** into your project.

```bash
go get github.com/rmhubbert/rmhttp
```

## Quickstart

The following code will get you up and running quickly with a basic GET endpoint.

```go
package main

import (
	"log"
	"net/http"

	"github.com/rmhubbert/rmhttp"
)

func myHandler := func(w http.ResponseWriter, r *http.Request) error {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Hello World"))
    return nil
}

func main() {
    // New() creates and intialises the app. You can optionally pass
    // in a configuration object.
    rmh := rmhttp.New()

    // Handle(), HandleFunc(), Post(), Put(), Patch(), Delete() and
    // Options() methods are also available.
    rmh.Get("/hello", myHandler)

    // Start() handles the server lifecycyle, including graceful
    // shutdown.
    log.Fatal(rmh.Start())
}
```

## Configuration

Configuration options can be set via environment variables or by passing in a Config object to the New() method, See [https://github.com/rmhubbert/rmhttp/blob/main/config.go](config.go) for details.

## Usage

**rmhttp** offers a fluent interface for building out your server functionality, allowing you to easily customise your server, groups, and routes. Here are some simple examples of the core functionality to get you started.

### Error Handling

You can register any custom or sentinel error with **rmhttp** via the RegisterError() method. This method takes the HTTP status code that you want to register and a variadic list of errors to register that status code for. Once registered, simply returning that error from your handler will trigger the associated status code to be returned alongside the error message.

```go
package main

import (
	"log"
	"net/http"

	"github.com/rmhubbert/rmhttp"
)

type CustomError struct{}

func (err CustomError) Error() string {
	return "custom 400 error"
}

var ErrMy400 = errors.New("my 400 error")

func myHandler := func(w http.ResponseWriter, r *http.Request) error {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Hello World"))
    return ErrMy400
}

func main() {
    rmh := rmhttp.New()
    // This handler will return a 400 HTTP status code.
    rmh.Get("/hello", myHandler)

    rmh.RegisterError(400, CustomError{}, ErrMy400)

    log.Fatal(rmh.Start())
}
```

### Groups

Routes can be easily grouped by registering them with a Group object. This allows all of the routes registered this way to inherit the group URL pattern plus any configured headers and middleware.

```go
package main

import (
	"log"
	"net/http"

	"github.com/rmhubbert/rmhttp"
)

func myHandler := func(w http.ResponseWriter, r *http.Request) error {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Hello World"))
    return nil
}

func main() {
    rmh := rmhttp.New()
    // The following creates a Group and then registers a Get route with that Group.
    // The route will be accessible at /api/hello.
    rmh.Group("/api").Get("/hello", myHandler)

    log.Fatal(rmh.Start())
}
```

### Headers

Headers can be easily added at the global, group, and route level by calling WithHeader() on the desired target.

```go
package main

import (
	"log"
	"net/http"

	"github.com/rmhubbert/rmhttp"
)

func myHandler := func(w http.ResponseWriter, r *http.Request) error {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Hello World"))
    return nil
}

func main() {
    rmh := rmhttp.New().WithHeader("X-Hello", "World")
    rmh.Get("/hello", myHandler).WithHeader("X-My", "Header")

    log.Fatal(rmh.Start())
}
```

### Middleware

Middleware can be easily added at the global, group, and route level by calling WithMiddleware() or Use() on the desired target.

```go
package main

import (
	"log"
	"net/http"

	"github.com/rmhubbert/rmhttp"
	"github.com/rmhubbert/rmhttp/middleware/recoverer"
)

func myHandler := func(w http.ResponseWriter, r *http.Request) error {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Hello World"))
    return nil
}

func main() {
    rmh := rmhttp.New().WithMiddleware(recoverer.Middleware())
    rmh.Get("/hello", myHandler)

    log.Fatal(rmh.Start())
}
```

## License

**rmhttp** is made available for use via the [MIT license](LICENSE).

## Contributing

Contributions are always welcome via [Pull Request](https://github.com/rmhubbert/rmhttp/pulls). Please make sure to add tests and make sure they are passing before submitting. It's also a good idea to lint your code with golint.

Contributors are expected to abide by the guidelines outlined in the [Contributor Covenant Code of Conduct](CONTRIBUTING.md)
