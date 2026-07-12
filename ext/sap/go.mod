module github.com/jhonsferg/traverse/ext/sap

go 1.25.0

require (
	github.com/jhonsferg/relay v0.4.5
	github.com/jhonsferg/traverse v0.1.0
	github.com/jhonsferg/traverse/ext/oauth2 v0.1.0
)

require (
	github.com/andybalholm/brotli v1.2.2 // indirect
	github.com/klauspost/compress v1.19.0 // indirect
	golang.org/x/net v0.57.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
)

replace (
	github.com/jhonsferg/traverse v0.1.0 => ../../
	github.com/jhonsferg/traverse/ext/oauth2 v0.1.0 => ../oauth2
)
