module github.com/larsartmann/httputil

go 1.26.5

require github.com/larsartmann/go-error-family v0.10.0

require golang.org/x/time v0.15.0

require github.com/justinas/nosurf v1.2.0

require github.com/larsartmann/httputil/server_timing v0.9.1

require github.com/larsartmann/go-etag v0.1.0

replace github.com/larsartmann/httputil/server_timing => ./server_timing
