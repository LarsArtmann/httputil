package httputil

import (
	"net/http"
	"strconv"
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
	// DenyUnmatched controls the behavior when a request origin does not
	// match any entry in AllowedOrigins and AllowAllOrigins is false. When
	// false (default), the middleware falls back to Access-Control-Allow-Origin: *,
	// allowing all origins. When true, no Access-Control-Allow-Origin header is
	// set, causing the browser to deny the cross-origin request.
	DenyUnmatched bool
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
		DenyUnmatched:      true,
	}
}

// Error codes for CORSConfig validation. All classified as Rejection:
// an invalid config is unacceptable input, retrying without changing it
// cannot succeed.
const (
	codeCorsCredentialsWithAllOrigins = Code("cors.credentials_with_all_origins")
	codeCorsMaxAgeNegative            = Code("cors.max_age_negative")
)

var (
	errCredentialsWithAllOrigins = codeCorsCredentialsWithAllOrigins.Rejection(
		"CORSConfig: AllowCredentials=true with AllowAllOrigins=true is not permitted by the CORS spec",
	)
	errNegativeMaxAge = codeCorsMaxAgeNegative.Rejection("CORSConfig: MaxAge must not be negative")
)

// Validate checks the CORSConfig for invalid combinations and returns an error
// describing the first issue found. A valid config can be used with CORS without
// causing browser-side CORS failures.
func (c CORSConfig) Validate() error {
	if c.AllowCredentials && c.AllowAllOrigins {
		return errCredentialsWithAllOrigins.
			WithContextAny("allow_credentials", c.AllowCredentials).
			WithContextAny("allow_all_origins", c.AllowAllOrigins)
	}

	if c.MaxAge < 0 {
		return errNegativeMaxAge.WithContextAny("max_age", c.MaxAge)
	}

	return nil
}

// CORS returns middleware that sets CORS headers based on the given config.
// Preflight OPTIONS requests receive a 204 No Content response unless
// OptionsPassthrough is set.
func CORS(cfg CORSConfig) Middleware {
	validateConfig("CORSConfig", cfg.Validate())

	allowCredentials := "false"
	if cfg.AllowCredentials {
		allowCredentials = "true"
	}

	// Pre-compute the joined header values once. cfg is captured per middleware
	// instance and never mutated, so recomputing these joins on every request
	// was pure waste (2-3 allocations per response).
	allowMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	exposeHeaders := strings.Join(cfg.ExposedHeaders, ", ")
	hasExposeHeaders := len(cfg.ExposedHeaders) > 0

	maxAge := ""
	if cfg.MaxAge > 0 {
		maxAge = strconv.Itoa(cfg.MaxAge)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			allowOrigin := "*"

			origin := req.Header.Get("Origin")
			if origin != "" {
				allowOrigin = resolveOrigin(origin, cfg)
			}

			if allowOrigin != "" {
				resp.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			}

			resp.Header().Set("Access-Control-Allow-Methods", allowMethods)
			resp.Header().Set("Access-Control-Allow-Headers", allowHeaders)
			resp.Header().Set("Access-Control-Allow-Credentials", allowCredentials)

			if hasExposeHeaders {
				resp.Header().Set("Access-Control-Expose-Headers", exposeHeaders)
			}

			if maxAge != "" {
				resp.Header().Set("Access-Control-Max-Age", maxAge)
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

	if cfg.DenyUnmatched {
		return ""
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
