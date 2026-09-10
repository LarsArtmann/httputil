package httputil

import (
	"io"
	"sync"
)

// errUnexpectedPoolType is returned when a writer pool yields an element
// that does not satisfy io.WriteCloser, meaning the pool and the factory
// disagree on the writer type. Classified as Infrastructure: this is a
// programming error in the WriterFactory contract, not a runtime condition.
var errUnexpectedPoolType = codeCompressionPoolTypeUnexpected.Infrastructure(
	"unexpected pool element type",
)

// codeCompressionPoolTypeUnexpected identifies a writer pool whose elements
// do not satisfy the io.WriteCloser contract of the factory that filled it.
const codeCompressionPoolTypeUnexpected = Code("compression.pool_type_unexpected")

// writerPool pairs a per-encoding sync.Pool with a construction-time probe of
// whether the factory's writers are resettable. Factories whose writers
// implement resettableWriter (gzip.Writer, flate.Writer) recycle through the
// pool; every other factory skips the pool entirely so custom non-resettable
// writers do not pay one wasted pooled allocation per request.
type writerPool struct {
	pool       *sync.Pool
	resettable bool
}

// newWriterPool probes the factory once (bound to io.Discard) to decide
// resettability, then builds the underlying pool whose New constructs fresh
// compression writers the same way. Callers acquire() a writer and use it
// bound to a concrete destination.
//
// The pool is owned by the negotiator for the lifetime of a single Compression
// middleware instance and keyed by encoding name, so it is bounded rather than
// process-global. A global registry keyed by the factory value is impossible
// (function values are not comparable in Go) and keying by the factory's
// parameter address created a fresh entry on every call (a leak).
//
// Panics when the factory fails to construct a writer: the factory contract
// is an internal invariant (built-in factories never fail), so a failure is
// an unrecoverable wiring bug, not a runtime condition.
func newWriterPool(factory WriterFactory) *writerPool {
	probe, err := factory(io.Discard)
	if err != nil {
		panic("httputil: writer factory failed: " + err.Error())
	}

	_, resettable := probe.(resettableWriter)

	return &writerPool{
		pool: &sync.Pool{
			New: func() any {
				// Discard writer; will be Reset() before use.
				w, err := factory(io.Discard)
				if err != nil {
					panic("httputil: writer factory failed: " + err.Error())
				}

				return w
			},
		},
		resettable: resettable,
	}
}

// acquire returns a compression writer bound to dst. Resettable factories
// draw from the pool and Reset the recycled writer to dst; every other
// factory builds a fresh writer directly, because a writer without
// Reset(io.Writer) cannot be rebound and pooling it would only add a wasted
// allocation per request.
func (p *writerPool) acquire(dst io.Writer, factory WriterFactory) (io.WriteCloser, error) {
	if !p.resettable {
		return factory(dst)
	}

	raw := p.pool.Get()

	writer, ok := raw.(io.WriteCloser)
	if !ok {
		return nil, errUnexpectedPoolType.WithContextf("pool_element_type", "%T", raw)
	}

	writer.(resettableWriter).Reset(dst)

	return writer, nil
}

// release returns a compression writer to the pool after a successful Close.
// Writers without resettableWriter support are dropped for garbage
// collection, mirroring acquire's pool skip for non-resettable factories.
func (p *writerPool) release(writer io.WriteCloser) {
	resettable, ok := writer.(resettableWriter)
	if !ok {
		return
	}

	p.pool.Put(resettable)
}
