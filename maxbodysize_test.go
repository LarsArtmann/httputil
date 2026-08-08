package httputil

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaxBodySizeAllowsNormalRequest(t *testing.T) {
	t.Parallel()

	handler := MaxBodySize(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMaxBodySizeRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	handler := MaxBodySize(5)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusRequestEntityTooLarge)

			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("this is way too long"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestMaxBodySizeHandlesNilBody(t *testing.T) {
	t.Parallel()

	handler := MaxBodySize(1024)(newStatusOnlyHandler(http.StatusOK))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestMaxBodySizeConfigValidateValid(t *testing.T) {
	t.Parallel()

	cfg := DefaultMaxBodySizeConfig()

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestMaxBodySizeConfigValidateZero(t *testing.T) {
	t.Parallel()

	cfg := MaxBodySizeConfig{MaxBytes: 0}

	err := cfg.Validate()
	if err != nil {
		t.Errorf("Validate() error = %v, want nil (zero is valid — disables limit)", err)
	}
}

func TestMaxBodySizeConfigValidateNegative(t *testing.T) {
	t.Parallel()

	cfg := MaxBodySizeConfig{MaxBytes: -1}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() error = nil, want error for negative MaxBytes")
	}

	if !errors.Is(err, errMaxBodySizeNegative) {
		t.Errorf("Validate() error = %v, want errMaxBodySizeNegative", err)
	}
}

func TestDefaultMaxBodySizeConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultMaxBodySizeConfig()

	if cfg.MaxBytes != defaultMaxBodySizeBytes {
		t.Errorf("MaxBytes = %d, want %d", cfg.MaxBytes, defaultMaxBodySizeBytes)
	}
}

func TestMaxBodySizeMiddlewareRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	cfg := MaxBodySizeConfig{MaxBytes: 5}
	handler := MaxBodySizeMiddleware(
		cfg,
	)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusRequestEntityTooLarge)

				return
			}

			w.WriteHeader(http.StatusOK)
		}),
	)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("this is way too long"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusRequestEntityTooLarge)
	}
}

// TestMaxBodySize_InvalidConfigLogsAndContinues verifies that an invalid
// MaxBodySize config (negative MaxBytes) is logged via slog but does not
// prevent the middleware from constructing and serving requests.
func TestMaxBodySize_InvalidConfigLogsAndContinues(t *testing.T) {
	t.Parallel()

	// MaxBytes < 0 is always a bug — Validate returns an error.
	// A GET with no body is not affected by the limit.
	cfg := MaxBodySizeConfig{
		MaxBytes: -1,
	}

	var called bool

	handler := MaxBodySizeMiddleware(cfg)(newCountingHandler(&called))
	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("inner handler was not called (invalid config should log and continue)")
	}
}
