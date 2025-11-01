# Agent Guidelines for rmhttp

## Build, Lint & Test Commands
- **Run all tests**: `go test -race -vet=off -v ./...`
- **Run single test**: `go test -v -run TestName ./path/to/package`
- **Lint**: `golangci-lint run` (uses `.golangci.yml` config)
- **Build**: `go build ./...`
- **Format**: Uses `goimports` and `golines` formatters

## Code Style
- **Go version**: 1.25.1
- **Imports**: Use `goimports` for automatic sorting/grouping (stdlib, external, internal)
- **Testing**: Use `github.com/stretchr/testify/assert` for assertions
- **Naming**: Use camelCase for unexported, PascalCase for exported; descriptive names
- **Comments**: Exported functions/types must have godoc comments starting with the name
- **Error handling**: Use `HTTPError` type for HTTP errors with status codes; return errors, don't panic
- **Types**: Prefer explicit types; use standard library types where possible
- **Middleware**: Follow `func(http.Handler) http.Handler` pattern
- **Headers**: Use lowercase for patterns/paths; uppercase for HTTP methods
- **Test names**: Use `Test_FunctionName` format with underscores
- **Linters**: Code must pass staticcheck, gosec, govet, errcheck, ineffassign, unused, testifylint

## Project Structure
- Core library in root (`rmhttp.go`, `route.go`, `router.go`, `server.go`, etc.)
- Middleware in `pkg/middleware/*/`
- E2E tests in `test/e2e/`
