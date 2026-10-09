module github.com/larsartmann/httputil

go 1.27

require (
	github.com/justinas/nosurf v1.2.0
	github.com/larsartmann/go-error-family v0.11.0
	github.com/larsartmann/go-etag/server v0.6.1
	github.com/larsartmann/httputil/server_timing v1.0.2
	golang.org/x/time v0.16.0
)

require github.com/larsartmann/go-etag/entitytag v0.6.1 // indirect

replace github.com/larsartmann/httputil/server_timing => ./server_timing
