package httputil

import (
	"errors"
	"net/http"
	"testing"
)

func TestRecovery_CatchesPanic(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	handler := Recovery(logger)(newPanicHandler("test panic"))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	assertStatus(t, rec, http.StatusInternalServerError)
}

func TestRecovery_PassesThroughNormal(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	called := false

	handler := Recovery(logger)(newCountingHandler(&called))

	req := newTestRequest(http.MethodGet, "/", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler was not called")
	}
}

// TestRecovery_RepanicsErrAbortHandler verifies that a handler panicking with
// the net/http sentinel http.ErrAbortHandler is NOT swallowed: the sentinel is
// re-panicked so the server's silent connection-abort handling applies instead
// of a doomed 500 write.
func TestRecovery_RepanicsErrAbortHandler(t *testing.T) {
	t.Parallel()

	logger := newTestLogger()
	aborting := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic(http.ErrAbortHandler)
	})

	handler := Recovery(logger)(aborting)

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("Recovery swallowed http.ErrAbortHandler, want it re-panicked")
		}

		err, ok := recovered.(error)
		if !ok || !errors.Is(err, http.ErrAbortHandler) {
			t.Fatalf("recovered %v, want http.ErrAbortHandler", recovered)
		}
	}()

	handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/", ""))
}

func TestRecoveryWithConfig_OnPanicReceivesRequestAndValue(t *testing.T) {
	t.Parallel()

	var (
		gotReq   *http.Request
		gotValue any
	)

	handler := RecoveryWithConfig(RecoveryConfig{
		Logger: newTestLogger(),
		OnPanic: func(r *http.Request, recovered any) {
			gotReq = r
			gotValue = recovered
		},
	})(newPanicHandler("hook probe"))

	req := newTestRequest(http.MethodGet, "/probe", "")
	rec := newRecorder()

	handler.ServeHTTP(rec, req)

	if gotReq != req {
		t.Errorf("hook request = %p, want %p", gotReq, req)
	}

	if gotValue != "hook probe" {
		t.Errorf("hook recovered value = %v, want %q", gotValue, "hook probe")
	}

	assertStatus(t, rec, http.StatusInternalServerError)
	assertBody(t, rec, "Internal Server Error")
}

func TestRecoveryWithConfig_NilHookMatchesRecovery(t *testing.T) {
	t.Parallel()

	handler := RecoveryWithConfig(RecoveryConfig{Logger: newTestLogger()})(newPanicHandler("nil hook probe"))

	rec := newRecorder()
	handler.ServeHTTP(rec, newTestRequest(http.MethodGet, "/", ""))

	assertStatus(t, rec, http.StatusInternalServerError)
	assertBody(t, rec, "Internal Server Error")
}

func TestRecoveryWithConfig_PanickingHookIsSwallowed(t *testing.T) {
	t.Parallel()

	handler := RecoveryWithConfig(RecoveryConfig{
		Logger: newTestLogger(),
		OnPanic: func(_ *http.Request, _ any) {
			panic("hook panic")
		},
	})(newPanicHandler("original panic"))

	rec := newRecorder()
	handler.ServeHTTP(rec, newTestRequest(http.MethodGet, "/", ""))

	assertStatus(t, rec, http.StatusInternalServerError)
	assertBody(t, rec, "Internal Server Error")
}

//nolint:paralleltest // swaps the global default logger; cannot run in parallel
func TestRecoveryWithConfig_NilLoggerFallsBackToSlogDefault(t *testing.T) {
	attrs := captureValidateConfigLog(t, func() {
		handler := RecoveryWithConfig(RecoveryConfig{})(newPanicHandler("nil logger probe"))
		handler.ServeHTTP(newRecorder(), newTestRequest(http.MethodGet, "/nil-logger", ""))
	})

	if attrs["msg"] != "panic recovered" {
		t.Errorf("log msg = %v, want %q", attrs["msg"], "panic recovered")
	}
}

func BenchmarkRecovery(b *testing.B) {
	logger := newTestLogger()
	middleware := Recovery(logger)

	inner := newNoOpHandler()
	handler := middleware(inner)
	req := newTestRequest(http.MethodGet, "/", "")

	for b.Loop() {
		rec := newRecorder()
		handler.ServeHTTP(rec, req)
	}
}
