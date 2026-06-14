package httputil

import (
	"errors"
	"io"
	"sync"
)

var errUnexpectedPoolType = errors.New("unexpected pool element type")

//nolint:gochecknoglobals // Per-factory writer pools amortize compression setup.
var (
	writerPoolsMu sync.RWMutex
	writerPools   = make(map[*WriterFactory]*sync.Pool)
)

func getWriterPool(factory WriterFactory) *sync.Pool {
	writerPoolsMu.RLock()

	pool, ok := writerPools[&factory]

	writerPoolsMu.RUnlock()

	if ok {
		return pool
	}

	writerPoolsMu.Lock()
	defer writerPoolsMu.Unlock()

	pool, ok = writerPools[&factory]
	if ok {
		return pool
	}

	pool = &sync.Pool{
		New: func() any {
			// Discard writer; will be Reset() before use.
			w, err := factory(io.Discard)
			if err != nil {
				panic("httputil: writer factory failed: " + err.Error())
			}

			return w
		},
	}
	writerPools[&factory] = pool

	return pool
}
