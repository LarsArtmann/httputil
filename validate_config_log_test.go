package httputil

import (
	"bytes"
	"encoding/json/v2"
	"log/slog"
	"testing"
)

// captureValidateConfigLog runs fn with slog's default logger swapped for a
// JSON handler writing to a buffer, and returns the decoded log record attrs.
// It exists because validateConfig logs via the package-level slog default.
func captureValidateConfigLog(t *testing.T, fn func()) map[string]any {
	t.Helper()

	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	fn()

	var record struct {
		Level string         `json:"level"`
		Msg   string         `json:"msg"`
		Attrs map[string]any `json:"-"`
	}

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) == 0 || len(lines[0]) == 0 {
		t.Fatal("no log output captured")
	}

	var decoded map[string]any

	err := json.Unmarshal(lines[0], &decoded)
	if err != nil {
		t.Fatalf("decoding log output %q: %v", lines[0], err)
	}

	record.Attrs = decoded

	return decoded
}

//nolint:paralleltest // swaps the global default logger; cannot run in parallel
func TestValidateConfigLogsCodeFamilyAndDomain(t *testing.T) {
	attrs := captureValidateConfigLog(t, func() {
		validateConfig("CORSConfig", errNegativeMaxAge.WithContextAny("max_age", -5))
	})

	if attrs["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR", attrs["level"])
	}

	if attrs["code"] != "cors.max_age_negative" {
		t.Errorf("code = %v, want cors.max_age_negative", attrs["code"])
	}

	if attrs["family"] != "rejection" {
		t.Errorf("family = %v, want rejection", attrs["family"])
	}

	if attrs["domain"] != "cors" {
		t.Errorf("domain = %v, want cors", attrs["domain"])
	}
}

//nolint:paralleltest // swaps the global default logger; cannot run in parallel
func TestValidateConfigLogsUncodedErrorWithoutCodeField(t *testing.T) {
	attrs := captureValidateConfigLog(t, func() {
		validateConfig("ExampleConfig", errTestPlain)
	})

	if _, has := attrs["code"]; has {
		t.Errorf("code field present for unclassified error: %v", attrs["code"])
	}

	if _, has := attrs["family"]; has {
		t.Errorf("family field present for unclassified error: %v", attrs["family"])
	}
}

//nolint:paralleltest // swaps the global default logger; cannot run in parallel
func TestValidateConfigNilErrorLogsNothing(t *testing.T) {
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))

	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	validateConfig("CORSConfig", nil)

	if buf.Len() != 0 {
		t.Errorf("log output = %q, want empty", buf.String())
	}
}
