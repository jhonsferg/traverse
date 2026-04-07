module github.com/jhonsferg/traverse

go 1.24.0

require github.com/jhonsferg/relay v0.2.0

require (
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	golang.org/x/net v0.50.0 // indirect
)

require (
	github.com/jhonsferg/traverse/ext/cache/memory v0.1.0
	golang.org/x/sync v0.19.0 // indirect
)

replace github.com/jhonsferg/traverse/ext/cache/memory => ./ext/cache/memory

replace github.com/jhonsferg/traverse/ext/cache => ./ext/cache

replace github.com/jhonsferg/traverse/ext/cache/redis => ./ext/cache/redis

replace github.com/jhonsferg/traverse/ext/graphql => ./ext/graphql

replace github.com/jhonsferg/traverse/ext/oauth2 => ./ext/oauth2

replace github.com/jhonsferg/traverse/ext/prometheus => ./ext/prometheus

replace github.com/jhonsferg/traverse/ext/sap => ./ext/sap

replace github.com/jhonsferg/traverse/ext/tracing => ./ext/tracing
