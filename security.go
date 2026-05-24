package httputil

import "net/http"

// SecurityHeadersConfig holds the configuration for security response headers.
type SecurityHeadersConfig struct {
	ContentTypeNosniff      bool
	FrameOptions            string
	XSSProtection           bool
	StrictTransportSecurity string
	ReferrerPolicy          string
	ContentSecurityPolicy   string
}

// DefaultSecurityHeadersConfig returns a SecurityHeadersConfig with sensible
// production defaults.
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentTypeNosniff:      true,
		FrameOptions:            "DENY",
		XSSProtection:           true,
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		ContentSecurityPolicy:   "",
		StrictTransportSecurity: "",
	}
}

// SecurityHeaders returns middleware that sets common security response headers
// based on the given configuration.
func SecurityHeaders(cfg SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if cfg.ContentTypeNosniff {
				resp.Header().Set("X-Content-Type-Options", "nosniff")
			}

			if cfg.FrameOptions != "" {
				resp.Header().Set("X-Frame-Options", cfg.FrameOptions)
			}

			if cfg.XSSProtection {
				resp.Header().Set("X-XSS-Protection", "0")
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
