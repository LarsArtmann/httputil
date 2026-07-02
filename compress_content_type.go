package httputil

import "strings"

// DefaultIncompressibleTypes returns the default list of content-type prefixes
// that should not be compressed (images, video, audio, pre-compressed archives).
// Use this to extend rather than replace the defaults in CompressionConfig.IncompressibleTypes.
func DefaultIncompressibleTypes() []string {
	return []string{
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
}

func isCompressibleContentType(contentType string, skipPrefixes []string) bool {
	if contentType == "" {
		return true
	}

	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(contentType, prefix) {
			return false
		}
	}

	return true
}
