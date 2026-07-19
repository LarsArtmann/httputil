package httputil

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

type mockMetricsRecorder struct {
	mu      sync.Mutex
	entries []mockMetricsEntry
}

type mockMetricsEntry struct {
	method   string
	path     string
	status   int
	duration time.Duration
}

func (m *mockMetricsRecorder) Record(method, path string, status int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.entries = append(m.entries, mockMetricsEntry{
		method:   method,
		path:     path,
		status:   status,
		duration: duration,
	})
}

func TestMetricsRecordsRequestData(t *testing.T) {
	t.Parallel()

	recorder := &mockMetricsRecorder{}
	cfg := DefaultMetricsConfig()
	cfg.Recorder = recorder

	handler := Metrics(cfg)(newStatusOnlyHandler(http.StatusTeapot))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	if len(recorder.entries) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recorder.entries))
	}

	entry := recorder.entries[0]

	if entry.method != http.MethodGet {
		t.Errorf("method = %q, want %q", entry.method, http.MethodGet)
	}

	if entry.path != "/test" {
		t.Errorf("path = %q, want %q", entry.path, "/test")
	}

	if entry.status != http.StatusTeapot {
		t.Errorf("status = %d, want %d", entry.status, http.StatusTeapot)
	}

	if entry.duration < 0 {
		t.Errorf("duration = %v, should be non-negative", entry.duration)
	}
}

func TestMetricsNormalizesStatusZero(t *testing.T) {
	t.Parallel()

	recorder := &mockMetricsRecorder{}
	cfg := DefaultMetricsConfig()
	cfg.Recorder = recorder

	handler := Metrics(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Don't call WriteHeader — ResponseRecorder defaults to 0
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	recorder.mu.Lock()
	defer recorder.mu.Unlock()

	if len(recorder.entries) != 1 {
		t.Fatalf("expected 1 record, got %d", len(recorder.entries))
	}

	if recorder.entries[0].status != http.StatusOK {
		t.Errorf(
			"status = %d, want %d (0 should normalize to 200)",
			recorder.entries[0].status,
			http.StatusOK,
		)
	}
}

func TestMetricsConfigValidateRejectsNilRecorder(t *testing.T) {
	t.Parallel()

	cfg := DefaultMetricsConfig()

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for nil Recorder, got nil")
	}
}
