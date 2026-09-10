package httputil

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"sync"
	"testing"
)

// newPoisonedWriterPool returns a writerPool whose probe reports resettable
// but whose pool yields nonWriter values, forcing the pool type-assertion
// failure path in acquire deterministically (no reliance on sync.Pool
// retention).
func newPoisonedWriterPool() *writerPool {
	return &writerPool{
		pool:       &sync.Pool{New: func() any { return &nonWriter{} }},
		resettable: true,
	}
}

// errFactoryMustNotBeCalled signals an acquire path that must never reach
// the caller-supplied factory.
var errFactoryMustNotBeCalled = errors.New("factory must not be called")

// plainWriteCloser is a minimal io.WriteCloser without Reset support,
// exercising the non-resettable factory path of writerPool.
type plainWriteCloser struct {
	io.Writer
}

func (w *plainWriteCloser) Close() error { return nil }

// countingPlainFactory returns a factory producing plainWriteCloser values
// and recording every invocation in calls.
func countingPlainFactory(calls *int) WriterFactory {
	return func(dst io.Writer) (io.WriteCloser, error) {
		*calls++

		return &plainWriteCloser{Writer: dst}, nil
	}
}

// TestWriterPool_NonResettableFactory_SkipsPool verifies that a factory whose
// writers lack Reset(io.Writer) never touches the sync.Pool: every acquire
// builds a fresh writer directly, so custom non-resettable factories pay no
// wasted pooled allocation per request.
func TestWriterPool_NonResettableFactory_SkipsPool(t *testing.T) {
	t.Parallel()

	var calls int

	factory := countingPlainFactory(&calls)
	pool := newWriterPool(factory) // construction probe: one call

	first, err := pool.acquire(io.Discard, factory)
	if err != nil {
		t.Fatalf("first acquire() error = %v, want nil", err)
	}

	second, err := pool.acquire(io.Discard, factory)
	if err != nil {
		t.Fatalf("second acquire() error = %v, want nil", err)
	}

	if calls != 3 {
		t.Errorf("factory call count = %d, want 3 (probe + two fresh acquires)", calls)
	}

	if first == second {
		t.Error("acquire returned the same writer instance twice, want distinct instances")
	}
}

// markerPoolWriter is a resettable io.WriteCloser returned by a fake pool
// New, making the resettable (pooled) acquire path deterministic in tests.
type markerPoolWriter struct {
	io.Writer
}

func (w *markerPoolWriter) Close() error { return nil }

func (w *markerPoolWriter) Reset(dst io.Writer) { w.Writer = dst }

// TestWriterPool_ResettableFactory_AcquireUsesPool verifies that a resettable
// writerPool resolves acquire through the sync.Pool (its New) rather than the
// caller-supplied factory.
func TestWriterPool_ResettableFactory_AcquireUsesPool(t *testing.T) {
	t.Parallel()

	pool := &writerPool{
		pool:       &sync.Pool{New: func() any { return &markerPoolWriter{} }},
		resettable: true,
	}

	writer, err := pool.acquire(io.Discard, func(io.Writer) (io.WriteCloser, error) {
		t.Error("factory called on the resettable (pooled) acquire path")

		return nil, errFactoryMustNotBeCalled
	})
	if err != nil {
		t.Fatalf("acquire() error = %v, want nil", err)
	}

	if _, ok := writer.(*markerPoolWriter); !ok {
		t.Errorf("acquire returned %T, want *markerPoolWriter from the pool", writer)
	}
}

// TestWriterPool_Release_NonResettable_IsNotStored verifies that releasing a
// writer without Reset support never stores it in the pool: the next acquire
// must produce a pool-native writer instead of the dropped instance.
func TestWriterPool_Release_NonResettable_IsNotStored(t *testing.T) {
	t.Parallel()

	pool := &writerPool{
		pool:       &sync.Pool{New: func() any { return &markerPoolWriter{} }},
		resettable: true,
	}

	released := &plainWriteCloser{Writer: io.Discard}
	pool.release(released)

	writer, err := pool.acquire(io.Discard, nil)
	if err != nil {
		t.Fatalf("acquire() error = %v, want nil", err)
	}

	if _, ok := writer.(*plainWriteCloser); ok {
		t.Error("acquire returned the released non-resettable writer, want a pool writer")
	}
}

// TestWriterPool_Acquire_ResetsWriterToDestination proves the recycled writer
// is actually Reset to the caller's destination: the probe writer is bound to
// io.Discard, so without the Reset the destination buffer would stay empty.
func TestWriterPool_Acquire_ResetsWriterToDestination(t *testing.T) {
	t.Parallel()

	factory := GzipWriterFactory(gzip.DefaultCompression)
	pool := newWriterPool(factory)

	var buf bytes.Buffer

	writer, err := pool.acquire(&buf, factory)
	if err != nil {
		t.Fatalf("acquire() error = %v, want nil", err)
	}

	payload := []byte("pooled writer must be reset to the caller destination")

	if _, err := writer.Write(payload); err != nil {
		t.Fatalf("Write() error = %v, want nil", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v, want nil", err)
	}

	zr, err := gzip.NewReader(&buf)
	if err != nil {
		t.Fatalf("gzip.NewReader() error = %v, want nil", err)
	}

	decoded, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gzip decode error = %v, want nil", err)
	}

	if !bytes.Equal(decoded, payload) {
		t.Errorf(
			"decoded body = %q (%d bytes), want %q (%d bytes)",
			decoded,
			len(decoded),
			payload,
			len(payload),
		)
	}
}

// TestWriterPool_ProbeDetectsResettableWriters pins the construction-time
// probe decision: stdlib gzip writers are resettable (poolable), the
// identity passthrough writer is not.
func TestWriterPool_ProbeDetectsResettableWriters(t *testing.T) {
	t.Parallel()

	gzipPool := newWriterPool(GzipWriterFactory(gzip.DefaultCompression))
	if !gzipPool.resettable {
		t.Error("gzip writer pool resettable = false, want true")
	}

	passthroughPool := newWriterPool(passthroughFactory)
	if passthroughPool.resettable {
		t.Error("passthrough writer pool resettable = true, want false")
	}
}
