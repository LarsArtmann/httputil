module github.com/larsartmann/httputil

go 1.27

require (
	github.com/justinas/nosurf v1.2.0
	github.com/larsartmann/go-error-family v0.10.1
	github.com/larsartmann/go-etag v0.5.0
	github.com/larsartmann/httputil/server_timing v1.0.1
	golang.org/x/time v0.16.0
)

replace github.com/larsartmann/httputil/server_timing => ./server_timing
