package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDetectCapabilitiesRecorderHasFlusherOnly(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	caps := DetectCapabilities(rec)

	if caps.Hijacker {
		t.Error("expected Hijacker=false for httptest.ResponseRecorder")
	}

	if !caps.Flusher {
		t.Error("expected Flusher=true for httptest.ResponseRecorder")
	}
}

func TestDetectCapabilitiesReportsFalseForBasicWriter(t *testing.T) {
	t.Parallel()

	w := &bareResponseWriter{}
	caps := DetectCapabilities(w)

	if caps.Hijacker {
		t.Error("expected Hijacker=false for bare writer")
	}

	if caps.Flusher {
		t.Error("expected Flusher=false for bare writer")
	}
}

func TestDetectCapabilitiesThroughResponseWrapper(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	wrapped := newResponseWrapper(rec)
	caps := DetectCapabilities(&wrapped)

	if !caps.Hijacker {
		t.Error("expected Hijacker=true through responseWrapper")
	}

	if !caps.Flusher {
		t.Error("expected Flusher=true through responseWrapper")
	}
}

type bareResponseWriter struct{}

func (w *bareResponseWriter) Header() http.Header {
	return http.Header{}
}

func (w *bareResponseWriter) Write(_ []byte) (int, error) {
	return 0, nil
}

func (w *bareResponseWriter) WriteHeader(_ int) {}
