module github.com/rmhubbert/rmhttp/v5

go 1.25.1

retract (
	v5.1.0 // Invalid module path - missing /v5 suffix
	v5.0.0 // Invalid module path - had /v4 instead of /v5
)

require (
	dario.cat/mergo v1.0.2
	github.com/caarlos0/env/v11 v11.3.1
	github.com/felixge/httpsnoop v1.0.4
	github.com/rs/cors v1.11.1
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
