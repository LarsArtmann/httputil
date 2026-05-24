package httputil

import "net/http"

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
		AllowedOrigins:   []string{"*"},
		AllowAllOrigins:  true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID"},
		ExposedHeaders:   []string{},
		AllowCredentials: false,
		MaxAge:           86400,
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
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				allowOrigin = resolveOrigin(origin, cfg)
			}

			w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
			w.Header().Set("Access-Control-Allow-Methods", join(cfg.AllowedMethods))
			w.Header().Set("Access-Control-Allow-Headers", join(cfg.AllowedHeaders))
			w.Header().Set("Access-Control-Allow-Credentials", allowCredentials)

			if len(cfg.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", join(cfg.ExposedHeaders))
			}

			if cfg.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", itoa(cfg.MaxAge))
			}

			if r.Method == http.MethodOptions && !cfg.OptionsPassthrough {
				w.WriteHeader(http.StatusNoContent)

				return
			}

			next.ServeHTTP(w, r)
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
