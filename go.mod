module github.com/rmhubbert/rmhttp/v5

go 1.26.0

retract (
	v5.1.0 // Invalid module path - missing /v5 suffix
	v5.0.0 // Invalid module path - had /v4 instead of /v5
)

require (
	dario.cat/mergo v1.0.2
	github.com/caarlos0/env/v11 v11.4.1
	github.com/felixge/httpsnoop v1.1.0
	github.com/grokify/mogo v0.74.7
	github.com/rs/cors v1.11.1
	github.com/stretchr/testify v1.12.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect
