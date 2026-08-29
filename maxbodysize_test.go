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

func TestMaxBodySizeContentLengthPassesThroughUnderLimit(t *testing.T) {
	t.Parallel()

	mw := MaxBodySizeMiddleware(MaxBodySizeConfig{MaxBytes: 1024})

	var seenContentLength int64 = -1
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seenContentLength = r.ContentLength
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello"))
	mw(handler).ServeHTTP(httptest.NewRecorder(), req)

	if seenContentLength != 5 {
		t.Errorf("ContentLength = %d, want 5 (unchanged under the limit)", seenContentLength)
	}
}

func TestMaxBodySizeReadBeyondLimitReturnsMaxBytesError(t *testing.T) {
	t.Parallel()

	mw := MaxBodySizeMiddleware(MaxBodySizeConfig{MaxBytes: 8})

	var tooMany *http.MaxBytesError
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if errors.As(err, &tooMany) {
			// expected
			return
		}

		t.Errorf("expected *http.MaxBytesError, got %v", err)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("way more than eight bytes"))
	mw(handler).ServeHTTP(httptest.NewRecorder(), req)

	if tooMany == nil {
		t.Fatal("expected the read to fail with *http.MaxBytesError")
	}

	if tooMany.Limit != 8 {
		t.Errorf("MaxBytesError.Limit = %d, want 8", tooMany.Limit)
	}
}

func TestMaxBodySizeKeepsBodyUntouchedForGET(t *testing.T) {
	t.Parallel()

	mw := MaxBodySizeMiddleware(MaxBodySizeConfig{MaxBytes: 1024})

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true

		if r.Body == nil {
			t.Error("GET request body must not be replaced with a nil reader")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mw(handler).ServeHTTP(httptest.NewRecorder(), req)

	if !called {
		t.Error("handler should have been called")
	}
}
