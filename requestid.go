package httputil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
)

type requestIDKey struct{}

const (
	requestIDBytes         = 16
	defaultRequestIDHeader = "X-Request-ID"
)

// RequestIDConfig holds the configuration for the request ID middleware.
type RequestIDConfig struct {
	HeaderName    string
	GenerateID    func() string
	ForwardHeader string
}

// DefaultRequestIDConfig returns a RequestIDConfig that reads the X-Request-ID
// header and falls back to a generated 128-bit random hex ID if not present.
func DefaultRequestIDConfig() RequestIDConfig {
	return RequestIDConfig{
		HeaderName:    defaultRequestIDHeader,
		ForwardHeader: defaultRequestIDHeader,
		GenerateID:    generateRequestID,
	}
}

var (
	errNilGenerateID   = errors.New("RequestIDConfig.GenerateID must not be nil")
	errEmptyHeaderName = errors.New("RequestIDConfig.HeaderName must not be empty")
	errEmptyForwardHdr = errors.New("RequestIDConfig.ForwardHeader must not be empty")
)

// Validate checks the RequestIDConfig for invalid values.
func (c RequestIDConfig) Validate() error {
	if c.GenerateID == nil {
		return errNilGenerateID
	}

	if c.HeaderName == "" {
		return errEmptyHeaderName
	}

	if c.ForwardHeader == "" {
		return errEmptyForwardHdr
	}

	return nil
}

// RequestID returns middleware that propagates or generates a request ID.
// The ID is stored in the request context and set as a response header.
func RequestID(cfg RequestIDConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			requestID := req.Header.Get(cfg.ForwardHeader)
			if requestID == "" {
				requestID = cfg.GenerateID()
			}

			resp.Header().Set(cfg.HeaderName, requestID)

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

func generateRequestID() string {
	buf := make([]byte, requestIDBytes)

	_, _ = rand.Read(buf)

	return hex.EncodeToString(buf)
}
