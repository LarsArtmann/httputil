package httputil

import (
	"errors"
	"net/http"
)

// Frame options values for SecurityHeadersConfig.FrameOptions.
// Per RFC 7034 §2.1, valid values are DENY, SAMEORIGIN, or absent
// (the header is omitted entirely).
const (
	frameOptionsDeny       = "DENY"
	frameOptionsSameOrigin = "SAMEORIGIN"

	// SecurityHeaderSkip is the sentinel value for suppressing a default
	// security header. Set ContentTypeOptions, FrameOptions, or
	// ReferrerPolicy to this value to omit that header entirely:
	//
	//	httputil.SecurityHeadersConfig{
	//	    ContentTypeOptions: httputil.SecurityHeaderSkip,
	//	    ContentSecurityPolicy: httputil.RecommendedCSP,
	//	}
	SecurityHeaderSkip = "-"

	// RecommendedHSTS is a recommended Strict-Transport-Security value for production.
	RecommendedHSTS = "max-age=31536000; includeSubDomains"

	// RecommendedCSP is a baseline Content-Security-Policy for HTMX applications.
	// Allows scripts from self (required for HTMX) and styles from self.
	RecommendedCSP = "default-src 'self'; script-src 'self'; style-src 'self'"
)

var errSecurityInvalidFrameOptions = errors.New(
	"SecurityHeadersConfig.FrameOptions must be DENY, SAMEORIGIN, SecurityHeaderSkip, or empty (default = no header)",
)

// SecurityHeadersConfig holds the configuration for security response headers.
type SecurityHeadersConfig struct {
	// ContentTypeNosniff sets X-Content-Type-Options to "nosniff" when true.
	// Legacy bool field — prefer ContentTypeOptions for full control.
	ContentTypeNosniff bool

	// ContentTypeOptions sets X-Content-Type-Options to an explicit value.
	// Takes precedence over ContentTypeNosniff. Set to SecurityHeaderSkip
	// to suppress the header.
	ContentTypeOptions string

	FrameOptions            string
	StrictTransportSecurity string
	ReferrerPolicy          string
	ContentSecurityPolicy   string

	// PermissionsPolicy sets the Permissions-Policy header.
	PermissionsPolicy string

	// Custom headers are applied after all other headers.
	Custom map[string]string
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
		ContentTypeOptions:      "",
		PermissionsPolicy:       "",
		Custom:                  nil,
	}
}

// Validate checks the SecurityHeadersConfig for invalid values. Returns nil
// if the config is usable, or a descriptive error otherwise.
//
// FrameOptions is the only field with constrained values: it must be empty
// (no header sent), "DENY", "SAMEORIGIN", or SecurityHeaderSkip ("-") —
// anything else produces an invalid X-Frame-Options header per RFC 7034 §2.1.
func (c SecurityHeadersConfig) Validate() error {
	switch c.FrameOptions {
	case "", frameOptionsDeny, frameOptionsSameOrigin, SecurityHeaderSkip:
		return nil
	default:
		return errSecurityInvalidFrameOptions
	}
}

// SecurityHeaders returns middleware that sets common security response headers
// based on the given configuration.
func SecurityHeaders(cfg SecurityHeadersConfig) Middleware {
	validateConfig("SecurityHeadersConfig", cfg.Validate())

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			// ContentTypeOptions (explicit string) takes precedence over
			// ContentTypeNosniff (legacy bool). SecurityHeaderSkip suppresses
			// the header entirely, even when ContentTypeNosniff is true.
			if cfg.ContentTypeOptions == SecurityHeaderSkip {
				// explicitly suppressed
			} else if cfg.ContentTypeOptions != "" {
				resp.Header().Set("X-Content-Type-Options", cfg.ContentTypeOptions)
			} else if cfg.ContentTypeNosniff {
				resp.Header().Set("X-Content-Type-Options", "nosniff")
			}

			if cfg.FrameOptions != "" && cfg.FrameOptions != SecurityHeaderSkip {
				resp.Header().Set("X-Frame-Options", cfg.FrameOptions)
			}

			if cfg.ReferrerPolicy != "" && cfg.ReferrerPolicy != SecurityHeaderSkip {
				resp.Header().Set("Referrer-Policy", cfg.ReferrerPolicy)
			}

			if cfg.ContentSecurityPolicy != "" {
				resp.Header().Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}

			if cfg.StrictTransportSecurity != "" {
				resp.Header().Set("Strict-Transport-Security", cfg.StrictTransportSecurity)
			}

			if cfg.PermissionsPolicy != "" {
				resp.Header().Set("Permissions-Policy", cfg.PermissionsPolicy)
			}

			for k, v := range cfg.Custom {
				resp.Header().Set(k, v)
			}

			next.ServeHTTP(resp, req)
		})
	}
}
