package httputil

import (
	"net/http"
	"strconv"
)

// ParseUintQuery extracts a base-10 unsigned integer from the named query
// parameter in the request URL. It returns 0 if the parameter is missing,
// empty, or not a valid unsigned integer within the 32-bit range.
//
// Callers that need clamped defaults (e.g. pagination) should pass the result
// through a constructor that applies sensible defaults for zero values.
func ParseUintQuery(r *http.Request, key string) uint {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0
	}

	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return 0
	}

	return uint(n)
}
