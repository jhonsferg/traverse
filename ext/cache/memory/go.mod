module github.com/jhonsferg/traverse/ext/cache/memory

go 1.25.0

require (
	github.com/jhonsferg/traverse v0.1.0
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/andybalholm/brotli v1.2.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jhonsferg/relay v0.3.1 // indirect
	github.com/klauspost/compress v1.19.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/jhonsferg/traverse v0.1.0 => ../../../
	github.com/jhonsferg/traverse/ext/cache v0.1.0 => ../
)
