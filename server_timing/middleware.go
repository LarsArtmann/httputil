package servertiming

import "net/http"

// Middleware wraps an http.Handler to intercept or modify request flow.
type Middleware func(http.Handler) http.Handler
