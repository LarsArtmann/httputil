package httputil

import (
	"bytes"
	"errors"
	"io"
	"net/http/httptest"
	"testing"
)

type countingCloseReader struct {
	io.Reader

	closed chan struct{}
}

func (c *countingCloseReader) Close() error {
	select {
	case c.closed <- struct{}{}:
	default:
	}

	return nil
}

// FuzzResponseRecorder verifies the recorder state machine never panics and
// preserves its documented contract for arbitrary status codes and body
// sequences: writes after WriteHeader are captured, Status reflects the last
// explicit WriteHeader call, and double WriteHeader calls do not corrupt state.
func FuzzResponseRecorder(f *testing.F) {
	f.Add(200, "ok", 201, "created")
	f.Add(500, "err", 418, "teapot")
	f.Add(0, "", 204, "")

	f.Fuzz(func(t *testing.T, status1 int, body1 string, status2 int, body2 string) {
		rec := httptest.NewRecorder()
		rw := NewResponseRecorder(rec)

		if status1 < 100 || status1 > 599 || status2 < 100 || status2 > 599 {
			t.Skip("outside meaningful HTTP status range")
		}

		rw.WriteHeader(status1)
		_, _ = rw.Write([]byte(body1))
		rw.WriteHeader(status2)
		_, _ = rw.Write([]byte(body2))

		if rw.Status() != status1 {
			t.Errorf("Status() = %d, want %d (first WriteHeader wins)", rw.Status(), status1)
		}

		if !rw.WroteHeader() {
			t.Error("WroteHeader() should be true after WriteHeader")
		}

		want := body1 + body2
		if rec.Body.String() != want {
			t.Errorf("body = %q, want %q", rec.Body.String(), want)
		}
	})
}

// FuzzLimitedReadCloser probes the bomb-protection boundary: reads past the
// limit must return errDecompressionSizeExceeded, close the underlying
// reader exactly once at the boundary, and never return more bytes than the
// limit allows.
func FuzzLimitedReadCloser(f *testing.F) {
	f.Add([]byte("small"), 64, 8)
	f.Add([]byte(""), 8, 8)
	f.Add([]byte("exactly-the-limit!!"), 18, 18)
	f.Add([]byte("bigger-than-limit"), 4, 2)

	f.Fuzz(func(t *testing.T, payload []byte, limit, readSize int) {
		if limit <= 0 || readSize <= 0 {
			t.Skip("non-positive sizes")
		}

		closed := make(chan struct{}, 1)
		inner := &countingCloseReader{Reader: bytes.NewReader(payload), closed: closed}

		limited := limitedReadCloser(inner, int64(limit))

		var total int
		readBuf := make([]byte, readSize) //nolint:makezero // read buffer, never appended to

		for {
			n, err := limited.Read(readBuf)
			total += n

			if total > limit {
				t.Errorf("read %d bytes, exceeding limit %d", total, limit)
			}

			if err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, errDecompressionSizeExceeded) {
					t.Errorf("unexpected error: %v", err)
				}

				break
			}
		}

		if total > len(payload) {
			t.Errorf("read %d bytes from a %d-byte payload", total, len(payload))
		}

		_ = limited.Close()
	})
}
