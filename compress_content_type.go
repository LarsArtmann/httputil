package httputil

import "strings"

//nolint:gochecknoglobals // Immutable reference data for content-type filtering.
var incompressiblePrefixes = []string{
	"image/",
	"video/",
	"audio/",
	"application/gzip",
	"application/zip",
	"application/pdf",
	"application/x-rar",
	"application/x-7z",
	"application/x-compress",
}

func isCompressibleContentType(contentType string) bool {
	if contentType == "" {
		return true
	}

	for _, prefix := range incompressiblePrefixes {
		if strings.HasPrefix(contentType, prefix) {
			return false
		}
	}

	return true
}
