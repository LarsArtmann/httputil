package httputil

import (
	"net/http"
	"testing"
)

// TestGenerateTimeOrderedID_Format verifies the ID is the expected length and hex-only.
func TestGenerateTimeOrderedID_Format(t *testing.T) {
	t.Parallel()

	generated := generateTimeOrderedID()

	if len(generated) != 32 {
		t.Errorf("ID length = %d, want 32", len(generated))
	}

	for _, c := range generated {
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'

		if !isDigit && !isLowerHex {
			t.Errorf("ID contains non-hex char %q in %q", c, generated)
		}
	}
}

// TestGenerateTimeOrderedID_Unique verifies consecutive IDs are distinct.
func TestGenerateTimeOrderedID_Unique(t *testing.T) {
	t.Parallel()

	const n = 1000

	seen := make(map[string]bool, n)

	for i := range n {
		generatedID := generateTimeOrderedID()
		if seen[generatedID] {
			t.Fatalf("duplicate ID after %d generations: %q", i, generatedID)
		}

		seen[generatedID] = true
	}
}

// TestGenerateTimeOrderedID_Sortable verifies IDs from the same second are
// monotonically increasing (counter portion increases).
func TestGenerateTimeOrderedID_Sortable(t *testing.T) {
	t.Parallel()

	first := generateTimeOrderedID()
	second := generateTimeOrderedID()

	// The counter is bytes 4-7 of the raw ID. As hex, these are chars 8-15
	// of the string (32 hex chars = 16 bytes). The first 8 hex chars
	// (bytes 0-3) are the timestamp.
	counterFirst := first[8:16]
	counterSecond := second[8:16]

	// In hex, the second counter must be greater-or-equal to the first
	// (it can wrap after 4B IDs/sec, vanishingly unlikely in test).
	if counterSecond < counterFirst {
		t.Errorf("counter went backwards: first=%q second=%q", counterFirst, counterSecond)
	}
}

// TestRequestID_DefaultGeneratesTimeOrderedID verifies the default
// GenerateID is a time-ordered ID with the expected format.
func TestRequestID_DefaultGeneratesTimeOrderedID(t *testing.T) {
	t.Parallel()

	cfg := DefaultRequestIDConfig()
	handler := RequestID(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())
		if len(id) != 32 {
			t.Errorf("context ID length = %d, want 32", len(id))
		}
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	got := rec.Header().Get("X-Request-ID")
	if len(got) != 32 {
		t.Errorf("response header X-Request-ID length = %d, want 32", len(got))
	}
}

// TestRequestID_ContextAndHeaderMatch verifies the context ID matches the response header.
func TestRequestID_ContextAndHeaderMatch(t *testing.T) {
	t.Parallel()

	cfg := DefaultRequestIDConfig()
	handler := RequestID(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextID := RequestIDFromContext(r.Context())

		headerID := w.Header().Get("X-Request-ID")
		if contextID != headerID {
			t.Errorf("context ID %q != header ID %q", contextID, headerID)
		}
	}))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)
}

// TestDrawRandomBytes_Concurrent verifies thread-safe operation.
func TestDrawRandomBytes_Concurrent(t *testing.T) {
	t.Parallel()

	const (
		goroutines = 100
		callsEach  = 100
	)

	done := make(chan struct{}, goroutines)

	for range goroutines {
		go func() {
			defer func() { done <- struct{}{} }()

			dst := make(
				[]byte,
				idRandBytes,
			) //nolint:makezero // pre-allocated buffer for crypto/rand.Read
			for range callsEach {
				drawRandomBytes(dst)
			}
		}()
	}

	for range goroutines {
		<-done
	}
}
