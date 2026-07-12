module github.com/jhonsferg/traverse/ext/cache/redis

go 1.25.0

require (
	github.com/jhonsferg/traverse v0.1.0
	github.com/redis/go-redis/v9 v9.21.0
	github.com/stretchr/testify v1.11.1
)

replace (
	github.com/jhonsferg/traverse v0.1.0 => ../../../
	github.com/jhonsferg/traverse/ext/cache v0.1.0 => ../
)

require (
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jhonsferg/relay v0.3.1 // indirect
	github.com/klauspost/compress v1.19.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
