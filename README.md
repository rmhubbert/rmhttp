# rmhttp

[![Go Reference](https://pkg.go.dev/badge/github.com/rmhubbert/rmhttp.svg)](https://pkg.go.dev/github.com/rmhubbert/rmhttp) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/rmhubbert/rmhttp?color=%23007D9C)
![GitHub Release Date](https://img.shields.io/github/release-date/rmhubbert/rmhttp?color=%23007D9C)
![GitHub commits since latest release](https://img.shields.io/github/commits-since/rmhubbert/rmhttp/latest?color=%23007D9C) [![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg?color=%23007D9C)](CONTRIBUTING.md)

**rmhttp** provides a lightweight wrapper around the Go standard library HTTP server and router provided by [net/http](https://pkg.go.dev/net/http) that allows for the easy addition of timeouts, groups, headers, and middleware (at the route, group and server level).

This package aims to make it easier to configure your routes and middleware, but then hand off as much as possible to the standard library once the server is running. Standard net/http handlers and middleware functions are used throughout.

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

func myHandler := func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Hello World"))
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
    log.Fatal(rmh.ListenAndServe())
}
```

## Configuration

Configuration options can be set via environment variables or by passing in a Config object to the New() method, See [https://github.com/rmhubbert/rmhttp/blob/main/config.go](config.go) for details.

## Usage

**rmhttp** offers a fluent interface for building out your server functionality, allowing you to easily customise your server, groups, and routes. Here are some simple examples of the core functionality to get you started.

### Error Handling

You can register your own handlers for 404 and 403 errors. These errors are normally triggered internally by http.ServeMux, and are normally not configurable.

```go
package main

import (
	"log"
	"net/http"

	"github.com/rmhubbert/rmhttp"
)

func my404Handler := func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte("Hello World"))
}

func main() {
    rmh := rmhttp.New()
    // This handler will replace the default 404 HTTP status code handler.
    rmh.StatusNotFoundHandler(my404Handler)

    log.Fatal(rmh.ListenAndServe())
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

func myHandler := func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Hello World"))
}

func main() {
    rmh := rmhttp.New()
    // The following creates a Group and then registers a Get route with that Group.
    // The route will be accessible at /api/hello.
    rmh.Group("/api").Get("/hello", myHandler)

    log.Fatal(rmh.ListenAndServe())
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

func myHandler := func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Hello World"))
}

func main() {
    rmh := rmhttp.New().WithHeader("X-Hello", "World")
    rmh.Get("/hello", myHandler).WithHeader("X-My", "Header")

    log.Fatal(rmh.ListenAndServe())
}
```

### Timeouts

Timeouts can be easily added at the global, group, and route level by calling WithTimeout() on the desired target. The length of timeout is set in seconds.

```go
package main

import (
	"log"
	"net/http"

	"github.com/rmhubbert/rmhttp"
)

func myHandler := func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Hello World"))
}

func main() {
    rmh := rmhttp.New().WithTimeout(5, "Global timeout message")
    rmh.Get("/hello", myHandler).WithTimeout(3, "Route timeout message")

    log.Fatal(rmh.ListenAndServe())
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

func myHandler := func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Hello World"))
}

func main() {
    rmh := rmhttp.New().WithMiddleware(recoverer.Middleware())
    rmh.Get("/hello", myHandler)

    log.Fatal(rmh.ListenAndServe())
}
```

## License

**rmhttp** is made available for use via the [MIT license](LICENSE).

## Contributing

Contributions are always welcome via [Pull Request](https://github.com/rmhubbert/rmhttp/pulls). Please make sure to add tests and make sure they are passing before submitting. It's also a good idea to lint your code with golintci-lint, using the config in this directory.

Contributors are expected to abide by the guidelines outlined in the [Contributor Covenant Code of Conduct](CONTRIBUTING.md)
