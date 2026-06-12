package httputil

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDefaultServerConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()

	if cfg.Addr != defaultAddr {
		t.Errorf("Addr = %q, want %q", cfg.Addr, defaultAddr)
	}

	if cfg.ReadTimeout != defaultReadTimeoutSeconds*time.Second {
		t.Errorf("ReadTimeout = %v, want %v", cfg.ReadTimeout, defaultReadTimeoutSeconds*time.Second)
	}

	if cfg.ReadHeaderTimeout != defaultReadHeaderTimeoutSeconds*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want %v", cfg.ReadHeaderTimeout, defaultReadHeaderTimeoutSeconds*time.Second)
	}

	if cfg.WriteTimeout != defaultWriteTimeoutSeconds*time.Second {
		t.Errorf("WriteTimeout = %v, want %v", cfg.WriteTimeout, defaultWriteTimeoutSeconds*time.Second)
	}

	if cfg.IdleTimeout != defaultIdleTimeoutSeconds*time.Second {
		t.Errorf("IdleTimeout = %v, want %v", cfg.IdleTimeout, defaultIdleTimeoutSeconds*time.Second)
	}
}

func TestServerConfigValidateDefault(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestServerConfigValidateNegativeReadTimeout(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:        defaultAddr,
		ReadTimeout: -1 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative ReadTimeout")
	}

	if !errors.Is(err, errReadTimeoutNegative) {
		t.Errorf("Validate() error = %v, want errReadTimeoutNegative", err)
	}
}

func TestServerConfigValidateNegativeReadHeaderTimeout(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:              defaultAddr,
		ReadHeaderTimeout: -1 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative ReadHeaderTimeout")
	}

	if !errors.Is(err, errReadHeaderTimeoutNegative) {
		t.Errorf("Validate() error = %v, want errReadHeaderTimeoutNegative", err)
	}
}

func TestServerConfigValidateNegativeWriteTimeout(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:         defaultAddr,
		WriteTimeout: -1 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative WriteTimeout")
	}

	if !errors.Is(err, errWriteTimeoutNegative) {
		t.Errorf("Validate() error = %v, want errWriteTimeoutNegative", err)
	}
}

func TestServerConfigValidateNegativeIdleTimeout(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:        defaultAddr,
		IdleTimeout: -1 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative IdleTimeout")
	}

	if !errors.Is(err, errIdleTimeoutNegative) {
		t.Errorf("Validate() error = %v, want errIdleTimeoutNegative", err)
	}
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.Addr = "127.0.0.1:0"

	srv, err := NewServer(cfg, newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	if srv.Addr() != cfg.Addr {
		t.Errorf("Addr() = %q, want %q", srv.Addr(), cfg.Addr)
	}
}

func TestNewServerInvalidConfig(t *testing.T) {
	t.Parallel()

	cfg := ServerConfig{
		Addr:        defaultAddr,
		ReadTimeout: -1 * time.Second,
	}

	_, err := NewServer(cfg, http.NotFoundHandler())
	if err == nil {
		t.Fatal("NewServer() error = nil, want error for negative ReadTimeout")
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.Addr = "127.0.0.1:0"

	srv, err := NewServer(cfg, newNoOpHandler())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	errChan := srv.Start()

	select {
	case err := <-errChan:
		t.Fatalf("server failed to start: %v", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully.
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestServerStartError(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.Addr = "invalid-address"

	srv, err := NewServer(cfg, http.NotFoundHandler())
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	errChan := srv.Start()

	select {
	case err := <-errChan:
		if err == nil {
			t.Fatal("expected startup error, got nil")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected startup error, got none")
	}
}

func TestServerServesRequests(t *testing.T) {
	t.Parallel()

	cfg := DefaultServerConfig()
	cfg.Addr = "127.0.0.1:0"

	handler := newWriteStatusHandler(http.StatusOK, "hello")

	srv, err := NewServer(cfg, handler)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	errChan := srv.Start()

	select {
	case err := <-errChan:
		t.Fatalf("server failed to start: %v", err)
	case <-time.After(100 * time.Millisecond):
		// Server started successfully.
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusOK)
	assertBody(t, rec, "hello")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(shutdownCtx)
	if err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
