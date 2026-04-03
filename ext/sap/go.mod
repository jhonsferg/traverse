module github.com/jhonsferg/traverse/ext/sap

go 1.24.0

require (
	github.com/jhonsferg/relay v0.1.3
	github.com/jhonsferg/traverse v0.1.0
	github.com/jhonsferg/traverse/ext/oauth2 v0.1.0
)

require golang.org/x/sync v0.16.0 // indirect

replace (
	github.com/jhonsferg/traverse v0.1.0 => ../../
	github.com/jhonsferg/traverse/ext/oauth2 v0.1.0 => ../oauth2
)
