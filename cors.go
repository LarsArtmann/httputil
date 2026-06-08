package httputil

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const defaultMaxAge = 86400 // seconds in 24 hours

// CORSConfig holds the configuration for CORS headers.
type CORSConfig struct {
	AllowedOrigins     []string
	AllowedMethods     []string
	AllowedHeaders     []string
	ExposedHeaders     []string
	AllowCredentials   bool
	MaxAge             int
	AllowAllOrigins    bool
	OptionsPassthrough bool
}

// DefaultCORSConfig returns a permissive development-friendly CORS config
// that allows all origins.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:  []string{"*"},
		AllowAllOrigins: true,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
			http.MethodOptions,
		},
		AllowedHeaders:     []string{headerContentType, "Authorization", defaultRequestIDHeader},
		ExposedHeaders:     []string{},
		AllowCredentials:   false,
		MaxAge:             defaultMaxAge,
		OptionsPassthrough: false,
	}
}

var (
	errCredentialsWithAllOrigins = errors.New(
		"CORSConfig: AllowCredentials=true with AllowAllOrigins=true is not permitted by the CORS spec",
	)
	errNegativeMaxAge = errors.New("CORSConfig: MaxAge must not be negative")
)

// Validate checks the CORSConfig for invalid combinations and returns an error
// describing the first issue found. A valid config can be used with CORS without
// causing browser-side CORS failures.
func (c CORSConfig) Validate() error {
	if c.AllowCredentials && c.AllowAllOrigins {
		return fmt.Errorf(
			"%w: browsers reject Access-Control-Allow-Origin: * when credentials are enabled",
			errCredentialsWithAllOrigins,
		)
	}

	if c.MaxAge < 0 {
		return fmt.Errorf("%w: got %d", errNegativeMaxAge, c.MaxAge)
	}

	return nil
}

// CORS returns middleware that sets CORS headers based on the given config.
// Preflight OPTIONS requests receive a 204 No Content response unless
// OptionsPassthrough is set.
func CORS(cfg CORSConfig) Middleware {
	allowCredentials := "false"
	if cfg.AllowCredentials {
		allowCredentials = "true"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			allowOrigin := "*"

			origin := req.Header.Get("Origin")
			if origin != "" {
				allowOrigin = resolveOrigin(origin, cfg)
			}

			resp.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			resp.Header().Set("Access-Control-Allow-Methods", join(cfg.AllowedMethods))
			resp.Header().Set("Access-Control-Allow-Headers", join(cfg.AllowedHeaders))
			resp.Header().Set("Access-Control-Allow-Credentials", allowCredentials)

			if len(cfg.ExposedHeaders) > 0 {
				resp.Header().Set("Access-Control-Expose-Headers", join(cfg.ExposedHeaders))
			}

			if cfg.MaxAge > 0 {
				resp.Header().Set("Access-Control-Max-Age", itoa(cfg.MaxAge))
			}

			if req.Method == http.MethodOptions && !cfg.OptionsPassthrough {
				resp.WriteHeader(http.StatusNoContent)

				return
			}

			next.ServeHTTP(resp, req)
		})
	}
}

func resolveOrigin(origin string, cfg CORSConfig) string {
	if cfg.AllowAllOrigins {
		return "*"
	}

	for _, allowed := range cfg.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return origin
		}

		if matchWildcardOrigin(allowed, origin) {
			return origin
		}
	}

	return "*"
}

// matchWildcardOrigin checks if an origin matches a wildcard pattern like
// "*.example.com". The wildcard only matches subdomains, not the domain itself.
func matchWildcardOrigin(pattern, origin string) bool {
	if !strings.HasPrefix(pattern, "*.") {
		return false
	}

	suffix := pattern[1:] // ".example.com"

	return strings.HasSuffix(origin, suffix)
}
