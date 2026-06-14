package httputil

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"strings"
	"testing"
)

// TestGzipWriterFactory_ReturnsUsableWriter verifies the factory creates a working writer.
func TestGzipWriterFactory_ReturnsUsableWriter(t *testing.T) {
	t.Parallel()

	var buf strings.Builder

	writer, err := GzipWriterFactory(gzip.DefaultCompression)(&buf)
	if err != nil {
		t.Fatalf("GzipWriterFactory() error = %v", err)
	}

	_, err = writer.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reader, err := gzip.NewReader(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatalf("gzip.NewReader error = %v", err)
	}
	defer func() { _ = reader.Close() }()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll error = %v", err)
	}

	if string(decoded) != "hello world" {
		t.Errorf("decoded = %q, want %q", decoded, "hello world")
	}
}

// TestDeflateWriterFactory_ReturnsUsableWriter verifies the factory creates a working writer.
func TestDeflateWriterFactory_ReturnsUsableWriter(t *testing.T) {
	t.Parallel()

	var buf strings.Builder

	writer, err := DeflateWriterFactory(gzip.DefaultCompression)(&buf)
	if err != nil {
		t.Fatalf("DeflateWriterFactory() error = %v", err)
	}

	_, err = writer.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reader := flate.NewReader(strings.NewReader(buf.String()))
	defer func() { _ = reader.Close() }()

	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll error = %v", err)
	}

	if string(decoded) != "hello world" {
		t.Errorf("decoded = %q, want %q", decoded, "hello world")
	}
}
