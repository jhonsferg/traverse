module github.com/jhonsferg/traverse/ext/cache/memory

go 1.24.0

require (
	github.com/jhonsferg/traverse v0.1.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jhonsferg/relay v0.1.3 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/jhonsferg/traverse v0.1.0 => ../../../
	github.com/jhonsferg/traverse/ext/cache v0.1.0 => ../
)
