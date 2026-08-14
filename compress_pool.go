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

// newWriterPool builds a sync.Pool whose New constructs fresh compression
// writers via factory bound to io.Discard. Callers Reset() the pooled writer
// to a concrete destination before use.
//
// The pool is owned by the negotiator for the lifetime of a single Compression
// middleware instance and keyed by encoding name, so it is bounded rather than
// process-global. A global registry keyed by the factory value is impossible
// (function values are not comparable in Go) and keying by the factory's
// parameter address created a fresh entry on every call (a leak).
func newWriterPool(factory WriterFactory) *sync.Pool {
	return &sync.Pool{
		New: func() any {
			// Discard writer; will be Reset() before use.
			w, err := factory(io.Discard)
			if err != nil {
				panic("httputil: writer factory failed: " + err.Error())
			}

			return w
		},
	}
}
