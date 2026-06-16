package httputil

import (
	"errors"
	"io"
	"sync"
)

var errUnexpectedPoolType = errors.New("unexpected pool element type")

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
