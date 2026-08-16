module github.com/larsartmann/httputil

go 1.26.5

require github.com/larsartmann/go-error-family v0.10.0

require golang.org/x/time v0.15.0

require github.com/justinas/nosurf v1.2.0

require github.com/larsartmann/httputil/server_timing v0.11.0

require github.com/larsartmann/go-etag v0.1.1

replace github.com/larsartmann/httputil/server_timing => ./server_timing

// Temporary until go-etag v0.2.0 is tagged: the server/ package lives on
// master only. Drop this replace and pin v0.2.0+ on release.
replace github.com/larsartmann/go-etag => ../go-etag
