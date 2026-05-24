package httputil

import "net/http"

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
		AllowedOrigins:     []string{"*"},
		AllowAllOrigins:    true,
		AllowedMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:     []string{"Content-Type", "Authorization", "X-Request-ID"},
		ExposedHeaders:     []string{},
		AllowCredentials:   false,
		MaxAge:             defaultMaxAge,
		OptionsPassthrough: false,
	}
}

// CORS returns middleware that sets CORS headers based on the given config.
// Preflight OPTIONS requests receive a 204 No Content response unless
// OptionsPassthrough is set.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	allowOrigin := "*"

	allowCredentials := "false"
	if cfg.AllowCredentials {
		allowCredentials = "true"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
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

	for _, o := range cfg.AllowedOrigins {
		if o == "*" || o == origin {
			return origin
		}
	}

	return "*"
}
