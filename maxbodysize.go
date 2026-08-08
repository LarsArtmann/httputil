package httputil

import (
	"errors"
	"fmt"
	"net/http"
)

const defaultMaxBodySizeBytes = 1 << 20 // 1 MiB

var errMaxBodySizeNegative = errors.New("MaxBodySizeConfig.MaxBytes must not be negative")

// MaxBodySizeConfig holds the configuration for the MaxBodySize middleware.
type MaxBodySizeConfig struct {
	MaxBytes int64
}

// DefaultMaxBodySizeConfig returns a MaxBodySizeConfig with a 1 MiB limit.
func DefaultMaxBodySizeConfig() MaxBodySizeConfig {
	return MaxBodySizeConfig{
		MaxBytes: defaultMaxBodySizeBytes,
	}
}

// Validate checks the MaxBodySizeConfig for invalid values. Returns nil if the
// config is usable, or a descriptive error identifying the first issue found.
//
// Validates:
//   - MaxBytes is non-negative (a negative limit is always a bug)
func (c MaxBodySizeConfig) Validate() error {
	if c.MaxBytes < 0 {
		return fmt.Errorf("%w: %d", errMaxBodySizeNegative, c.MaxBytes)
	}

	return nil
}

// MaxBodySizeMiddleware returns middleware that limits request body size using
// the provided configuration. Call [MaxBodySizeConfig.Validate] before
// constructing the middleware to surface configuration errors at startup.
func MaxBodySizeMiddleware(cfg MaxBodySizeConfig) Middleware {
	validateConfig("MaxBodySizeConfig", cfg.Validate())

	return MaxBodySize(cfg.MaxBytes)
}

// MaxBodySize returns middleware that limits request body size to maxBytes.
// When the limit is exceeded, the underlying read returns an error and the
// connection is closed to prevent the client from sending more data.
//
// Handlers should check for errors from r.Body.Read and respond with
// http.StatusRequestEntityTooLarge (413) as appropriate.
func MaxBodySize(maxBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}

			next.ServeHTTP(w, r)
		})
	}
}
