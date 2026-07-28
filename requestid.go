package httputil

import (
	"context"
	"errors"
	"net/http"
)

type requestIDKey struct{}

const defaultRequestIDHeader = "X-Request-ID"

// RequestIDConfig holds the configuration for the request ID middleware.
type RequestIDConfig struct {
	ResponseHeader string
	GenerateID     func() string
	IncomingHeader string
}

// DefaultRequestIDConfig returns a RequestIDConfig that reads the X-Request-ID
// header and falls back to a generated 32-character time-ordered hex ID if
// not present. The generated ID is sortable by creation time, monotonic
// within a second, and unique across the process.
func DefaultRequestIDConfig() RequestIDConfig {
	return RequestIDConfig{
		ResponseHeader: defaultRequestIDHeader,
		IncomingHeader: defaultRequestIDHeader,
		GenerateID:     generateTimeOrderedID,
	}
}

var (
	errNilGenerateID       = errors.New("RequestIDConfig.GenerateID must not be nil")
	errEmptyResponseHeader = errors.New("RequestIDConfig.ResponseHeader must not be empty")
	errEmptyIncomingHeader = errors.New("RequestIDConfig.IncomingHeader must not be empty")
)

// Validate checks the RequestIDConfig for invalid values.
func (c RequestIDConfig) Validate() error {
	if c.GenerateID == nil {
		return errNilGenerateID
	}

	if c.ResponseHeader == "" {
		return errEmptyResponseHeader
	}

	if c.IncomingHeader == "" {
		return errEmptyIncomingHeader
	}

	return nil
}

// RequestID returns middleware that propagates or generates a request ID.
// The ID is stored in the request context and set as a response header.
func RequestID(cfg RequestIDConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			requestID := req.Header.Get(cfg.IncomingHeader)
			if requestID == "" {
				requestID = cfg.GenerateID()
			}

			resp.Header().Set(cfg.ResponseHeader, requestID)

			ctx := context.WithValue(req.Context(), requestIDKey{}, requestID)
			next.ServeHTTP(resp, req.WithContext(ctx))
		})
	}
}

// RequestIDFromContext retrieves the request ID from the context.
// Returns empty string if no ID was stored.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey{}).(string)

	return requestID
}
