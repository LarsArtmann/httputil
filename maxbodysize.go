package httputil

import (
	"net/http"
)

const defaultMaxBodySizeBytes = 1 << 20 // 1 MiB

// codeMaxBodySizeNegative classifies a negative body-size limit as Rejection.
const codeMaxBodySizeNegative = Code("maxbodysize.max_bytes_negative")

var errMaxBodySizeNegative = codeMaxBodySizeNegative.Rejection(
	"MaxBodySizeConfig.MaxBytes must not be negative",
)

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
		return errMaxBodySizeNegative.WithContextAny("max_bytes", c.MaxBytes)
	}

	return nil
}

// MaxBodySizeMiddleware returns middleware that limits request body size using
// the provided configuration. The config is validated at construction time;
// invalid values are logged via slog and the middleware continues with the
// provided values.
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
