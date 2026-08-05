package httputil

import (
	"errors"
	"net/http"
)

// Frame options values for SecurityHeadersConfig.FrameOptions.
// Per RFC 7034 §2.1, valid values are DENY, SAMEORIGIN, or absent
// (the header is omitted entirely).
const (
	frameOptionsDeny      = "DENY"
	frameOptionsSameOrigin = "SAMEORIGIN"
)

var (
	errSecurityInvalidFrameOptions = errors.New(
		"SecurityHeadersConfig.FrameOptions must be DENY, SAMEORIGIN, or empty (default = no header)",
	)
)

// SecurityHeadersConfig holds the configuration for security response headers.
type SecurityHeadersConfig struct {
	ContentTypeNosniff      bool
	FrameOptions            string
	StrictTransportSecurity string
	ReferrerPolicy          string
	ContentSecurityPolicy   string
}

// DefaultSecurityHeadersConfig returns a SecurityHeadersConfig with sensible
// production defaults.
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentTypeNosniff:      true,
		FrameOptions:            frameOptionsDeny,
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		ContentSecurityPolicy:   "",
		StrictTransportSecurity: "",
	}
}

// Validate checks the SecurityHeadersConfig for invalid values. Returns nil
// if the config is usable, or a descriptive error otherwise.
//
// FrameOptions is the only field with constrained values: it must be empty
// (no header sent), "DENY", or "SAMEORIGIN" — anything else produces an
// invalid X-Frame-Options header per RFC 7034 §2.1.
func (c SecurityHeadersConfig) Validate() error {
	switch c.FrameOptions {
	case "", frameOptionsDeny, frameOptionsSameOrigin:
		return nil
	default:
		return errSecurityInvalidFrameOptions
	}
}

// SecurityHeaders returns middleware that sets common security response headers
// based on the given configuration.
func SecurityHeaders(cfg SecurityHeadersConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if cfg.ContentTypeNosniff {
				resp.Header().Set("X-Content-Type-Options", "nosniff")
			}

			if cfg.FrameOptions != "" {
				resp.Header().Set("X-Frame-Options", cfg.FrameOptions)
			}

			if cfg.ReferrerPolicy != "" {
				resp.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
			}

			if cfg.ContentSecurityPolicy != "" {
				resp.Header().Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}

			if cfg.StrictTransportSecurity != "" {
				resp.Header().Set("Strict-Transport-Security", cfg.StrictTransportSecurity)
			}

			next.ServeHTTP(resp, req)
		})
	}
}
