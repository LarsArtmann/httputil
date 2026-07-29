# Brotli / Zstd Compression Encoders

httputil ships with gzip and deflate support out of the box. Modern encodings like **brotli** and **zstd** are available through the `WriterFactory` plugin interface — no core dependency required.

## Pattern

Register a factory for each encoding you want to support. The factory receives an `io.Writer` (the underlying `http.ResponseWriter`) and returns an `io.WriteCloser` that compresses data written to it.

```go
// This example illustrates the WriterFactory pattern. It references an
// external compression library (e.g. github.com/andybalholm/brotli) that is
// NOT a dependency of httputil — add it to your go.mod to compile.
package main

import (
    "io"
    "net/http"

    "github.com/larsartmann/httputil"
    // "github.com/andybalholm/brotli"  // external dependency
)

// BrotliWriter is a minimal adapter. Replace with your library's writer.
type BrotliWriter struct {
    enc io.WriteCloser
}

func (w *BrotliWriter) Write(p []byte) (int, error) {
    return w.enc.Write(p)
}

func (w *BrotliWriter) Close() error {
    return w.enc.Close()
}

// ResettableBrotliWriter implements resettableWriter (optional but recommended
// for pool reuse — avoids allocating a new encoder per request).
type ResettableBrotliWriter struct {
    enc  io.WriteCloser // in real code: *brotli.Writer
    dst  io.Writer
    lvl  int
}

func (w *ResettableBrotliWriter) Write(p []byte) (int, error) {
    return w.enc.Write(p)
}

func (w *ResettableBrotliWriter) Close() error {
    return w.enc.Close()
}

// Reset allows the compression pool to reuse this writer across requests.
// Implementing this interface (an unexported `Reset(io.Writer)` method) makes
// the pool recycle the encoder instead of allocating a new one per request.
func (w *ResettableBrotliWriter) Reset(dst io.Writer) {
    w.dst = dst
    // w.enc = brotli.NewWriterLevel(dst, w.lvl) // external library constructor
}

func brotliFactory(level int) httputil.WriterFactory {
    return func(dst io.Writer) (io.WriteCloser, error) {
        return &ResettableBrotliWriter{
            // enc: brotli.NewWriterLevel(dst, level), // external library
            dst: dst,
            lvl: level,
        }, nil
    }
}

func main() {
    cfg := httputil.DefaultCompressionConfig()

    // Start from the built-in factories (gzip, deflate, identity).
    factories := httputil.DefaultWriterFactories()

    // Add brotli and zstd.
    factories["br"] = brotliFactory(4) // 4 = brotli.DefaultCompression
    // factories["zstd"] = zstdFactory(zstd.SpeedDefault)

    cfg.WriterFactories = factories

    mux := http.NewServeMux()
    handler := httputil.Compression(cfg)(mux)
    _ = handler
}
```

## Priority Order

httputil negotiates encodings by server priority when the client sends `Accept-Encoding` without quality values. The built-in priority is:

```
brotli > zstd > gzip > deflate > identity
```

Brotli and zstd are only negotiated if they are registered in `WriterFactories` and the client accepts them. If the client sends `Accept-Encoding: gzip, br;q=0.9`, gzip wins because the client downgraded brotli via q-value.

## Pool Reuse

Writers that implement the (unexported) `resettableWriter` interface — any type with a `Reset(io.Writer)` method — are recycled through a `sync.Pool` owned by the middleware. This avoids allocating a new encoder per request.

Gzip and deflate writers from the stdlib implement `Reset` natively. For custom encoders, add a `Reset` method as shown above.

## Adding a Custom Encoding

```go
cfg := httputil.DefaultCompressionConfig()
factories := httputil.DefaultWriterFactories()
factories["lz4"] = myLZ4Factory
cfg.WriterFactories = factories
```

The encoding name must match what the client sends in `Accept-Encoding` (e.g., `"br"`, `"zstd"`, `"lz4"`).
