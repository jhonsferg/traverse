module github.com/jhonsferg/traverse/ext/cache/memory

go 1.24.0

require (
	github.com/jhonsferg/traverse v0.1.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jhonsferg/relay v0.2.2 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.50.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/jhonsferg/traverse v0.1.0 => ../../../
	github.com/jhonsferg/traverse/ext/cache v0.1.0 => ../
)
