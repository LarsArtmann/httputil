package httputil

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"time"
)

const (
	defaultReadTimeoutSeconds       = 10
	defaultReadHeaderTimeoutSeconds = 5
	defaultWriteTimeoutSeconds      = 30
	defaultIdleTimeoutSeconds       = 60
	defaultShutdownTimeoutSeconds   = 30
	defaultAddr                     = ":8080"
)

// Error codes for ServerConfig validation. All config codes are classified
// as Rejection: an invalid config is unacceptable input.
const (
	codeServerAddrEmpty                 = Code("server.addr_empty")
	codeServerReadTimeoutNegative       = Code("server.read_timeout_negative")
	codeServerReadHeaderTimeoutNegative = Code("server.read_header_timeout_negative")
	codeServerWriteTimeoutNegative      = Code("server.write_timeout_negative")
	codeServerIdleTimeoutNegative       = Code("server.idle_timeout_negative")
	codeServerShutdownTimeoutNegative   = Code("server.shutdown_timeout_negative")
	codeServerTimeoutOrdering           = Code("server.timeout_ordering")
	codeServerTLSMinVersionInsecure     = Code("server.tls_min_version_insecure")
	codeServerShutdownFailed            = Code("server.shutdown_failed")
)

var (
	errReadTimeoutNegative = codeServerReadTimeoutNegative.Rejection(
		"ServerConfig.ReadTimeout must not be negative",
	)
	errReadHeaderTimeoutNegative = codeServerReadHeaderTimeoutNegative.Rejection(
		"ServerConfig.ReadHeaderTimeout must not be negative",
	)
	errWriteTimeoutNegative = codeServerWriteTimeoutNegative.Rejection(
		"ServerConfig.WriteTimeout must not be negative",
	)
	errIdleTimeoutNegative = codeServerIdleTimeoutNegative.Rejection(
		"ServerConfig.IdleTimeout must not be negative",
	)
	errShutdownTimeoutNegative = codeServerShutdownTimeoutNegative.Rejection(
		"ServerConfig.ShutdownTimeout must not be negative",
	)
	errServerAddrEmpty = codeServerAddrEmpty.Rejection(
		"ServerConfig.Addr must not be empty (e.g. \":8080\" or \":http\")",
	)
	errServerTimeoutOrdering = codeServerTimeoutOrdering.Rejection(
		"ServerConfig.ReadHeaderTimeout must be <= ReadTimeout (RFC 7230 §6)",
	)
	errTLSMinVersionInsecure = codeServerTLSMinVersionInsecure.Rejection(
		"ServerConfig.TLSConfig.MinVersion must be at least TLS 1.2 (RFC 8996)",
	)
	// errServerShutdownFailed wraps shutdown failures as Infrastructure: the
	// surrounding process state, not the request, is what failed.
	errServerShutdownFailed = codeServerShutdownFailed.Infrastructure("server shutdown failed")
)

// ServerConfig holds the configuration for an HTTP server.
type ServerConfig struct {
	Addr              string
	TLSConfig         *tls.Config
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

// DefaultServerConfig returns a ServerConfig with sensible production defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Addr:              defaultAddr,
		TLSConfig:         nil,
		ReadTimeout:       defaultReadTimeoutSeconds * time.Second,
		ReadHeaderTimeout: defaultReadHeaderTimeoutSeconds * time.Second,
		WriteTimeout:      defaultWriteTimeoutSeconds * time.Second,
		IdleTimeout:       defaultIdleTimeoutSeconds * time.Second,
		ShutdownTimeout:   defaultShutdownTimeoutSeconds * time.Second,
	}
}

// Validate checks the ServerConfig for invalid values. Returns nil if the
// config is usable, or a descriptive error identifying the first issue found.
//
// Validates:
//   - Addr is non-empty (empty Addr would bind to a random port — usually a bug)
//   - All timeouts are non-negative
//   - ReadHeaderTimeout <= ReadTimeout (the underlying http.Server enforces
//     this internally; checking here surfaces the misconfiguration clearly)
//   - TLSConfig.MinVersion (when set) is at least TLS 1.2 (RFC 8996 deprecates
//     TLS 1.0/1.1; a zero MinVersion defaults to TLS 1.2 in Go 1.18+, which is safe)
func (c ServerConfig) Validate() error {
	if c.Addr == "" {
		return errServerAddrEmpty
	}

	if c.ReadTimeout < 0 {
		return errReadTimeoutNegative.WithContextAny("read_timeout", c.ReadTimeout)
	}

	if c.ReadHeaderTimeout < 0 {
		return errReadHeaderTimeoutNegative.WithContextAny(
			"read_header_timeout",
			c.ReadHeaderTimeout,
		)
	}

	if c.WriteTimeout < 0 {
		return errWriteTimeoutNegative.WithContextAny("write_timeout", c.WriteTimeout)
	}

	if c.IdleTimeout < 0 {
		return errIdleTimeoutNegative.WithContextAny("idle_timeout", c.IdleTimeout)
	}

	if c.ShutdownTimeout < 0 {
		return errShutdownTimeoutNegative.WithContextAny("shutdown_timeout", c.ShutdownTimeout)
	}

	if c.ReadTimeout > 0 && c.ReadHeaderTimeout > c.ReadTimeout {
		return errServerTimeoutOrdering.
			WithContextAny("read_header_timeout", c.ReadHeaderTimeout).
			WithContextAny("read_timeout", c.ReadTimeout)
	}

	if c.TLSConfig != nil && c.TLSConfig.MinVersion != 0 &&
		c.TLSConfig.MinVersion < tls.VersionTLS12 {
		return errTLSMinVersionInsecure.WithContextf(
			"min_version",
			"0x%04x",
			c.TLSConfig.MinVersion,
		)
	}

	return nil
}

// Server wraps an http.Server with lifecycle helpers.
type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
}

// NewServer creates a new Server with the given configuration and handler.
// The handler is typically a *http.ServeMux or a middleware-wrapped handler.
func NewServer(cfg ServerConfig, handler http.Handler) (*Server, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}

	server := &Server{
		httpServer: &http.Server{
			Addr:                         cfg.Addr,
			Handler:                      handler,
			DisableGeneralOptionsHandler: false,
			TLSConfig:                    cfg.TLSConfig,
			ReadTimeout:                  cfg.ReadTimeout,
			ReadHeaderTimeout:            cfg.ReadHeaderTimeout,
			WriteTimeout:                 cfg.WriteTimeout,
			IdleTimeout:                  cfg.IdleTimeout,
			MaxHeaderBytes:               0,
			TLSNextProto:                 nil,
			ConnState:                    nil,
			ErrorLog:                     nil,
			BaseContext:                  nil,
			ConnContext:                  nil,
			HTTP2:                        nil,
			Protocols:                    nil,
		},
		shutdownTimeout: cfg.ShutdownTimeout,
	}

	return server, nil
}

// Start begins listening on the configured address in a goroutine.
// It returns a channel that receives any non-shutdown error from ListenAndServe.
func (srv *Server) Start() <-chan error {
	errChan := make(chan error, 1)

	go func() {
		err := srv.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	return errChan
}

// Shutdown gracefully shuts down the server. If the provided context has no
// deadline and the server was configured with a ShutdownTimeout, a timeout
// context is derived automatically to prevent indefinite hangs.
func (srv *Server) Shutdown(ctx context.Context) error {
	if _, ok := ctx.Deadline(); !ok && srv.shutdownTimeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, srv.shutdownTimeout)
		defer cancel()
	}

	err := srv.httpServer.Shutdown(ctx)
	if err != nil {
		return errServerShutdownFailed.WithCause(err)
	}

	return nil
}

// Addr returns the server's listen address.
func (srv *Server) Addr() string {
	return srv.httpServer.Addr
}
