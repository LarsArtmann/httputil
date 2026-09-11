package main

import (
	"bytes"
	"strings"
	"testing"
)

// sampleReport is a minimal but well-formed `go tool cover -func` report.
const sampleReport = `github.com/example/pkg/a.go:10:	funcA	80.0%
github.com/example/pkg/b.go:20:	funcB	100.0%
total:							(statements)	90.0%
`

func runWithReport(t *testing.T, args []string, report string) (int, string, string) {
	t.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code := run(args, strings.NewReader(report), stdout, stderr)

	return code, stdout.String(), stderr.String()
}

func TestRun_PassAboveThreshold(t *testing.T) {
	t.Parallel()

	code, stdout, stderr := runWithReport(t, []string{"85"}, sampleReport)

	if code != exitOK {
		t.Errorf("exit code = %d, want %d (stderr: %q)", code, exitOK, stderr)
	}

	if !strings.Contains(stdout, "total:") {
		t.Errorf("stdout = %q, want the echoed total line", stdout)
	}
}

func TestRun_PassAtExactThreshold(t *testing.T) {
	t.Parallel()

	code, _, stderr := runWithReport(t, []string{"90"}, sampleReport)

	if code != exitOK {
		t.Errorf("exit code = %d, want %d for exactly-at-threshold (stderr: %q)", code, exitOK, stderr)
	}
}

func TestRun_FailsBelowThreshold(t *testing.T) {
	t.Parallel()

	code, stdout, stderr := runWithReport(t, []string{"95"}, sampleReport)

	if code != exitBelowThreshold {
		t.Errorf("exit code = %d, want %d", code, exitBelowThreshold)
	}

	if !strings.Contains(stdout, "total:") {
		t.Errorf("stdout = %q, want the echoed total line even on failure", stdout)
	}

	if !strings.Contains(stderr, "::error::Coverage 90.00% is below 95.00% threshold") {
		t.Errorf("stderr = %q, want the GitHub Actions error annotation", stderr)
	}
}

func TestRun_MalformedReportFailsLoudly(t *testing.T) {
	t.Parallel()

	code, _, stderr := runWithReport(t, []string{"95"}, "line one\nline two\n")

	if code != exitUsage {
		t.Errorf("exit code = %d, want %d for a report with no total line", code, exitUsage)
	}

	if !strings.Contains(stderr, "no total: line found") {
		t.Errorf("stderr = %q, want the no-total-line error", stderr)
	}
}

func TestRun_EmptyReportFailsLoudly(t *testing.T) {
	t.Parallel()

	code, _, stderr := runWithReport(t, []string{"95"}, "")

	if code != exitUsage {
		t.Errorf("exit code = %d, want %d for an empty report", code, exitUsage)
	}

	if !strings.Contains(stderr, "no total: line found") {
		t.Errorf("stderr = %q, want the no-total-line error", stderr)
	}
}

func TestRun_UnparsableTotalLineFailsLoudly(t *testing.T) {
	t.Parallel()

	code, _, stderr := runWithReport(t, []string{"95"}, "total:\t(statements)\tninety percent\n")

	if code != exitUsage {
		t.Errorf("exit code = %d, want %d for an unparsable total percentage", code, exitUsage)
	}

	if !strings.Contains(stderr, "cannot parse coverage") {
		t.Errorf("stderr = %q, want the parse error", stderr)
	}
}

func TestRun_WrongArgumentCount(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{nil, {"95", "extra"}} {
		code, _, stderr := runWithReport(t, args, sampleReport)

		if code != exitUsage {
			t.Errorf("args %v: exit code = %d, want %d", args, code, exitUsage)
		}

		if !strings.Contains(stderr, "usage:") {
			t.Errorf("args %v: stderr = %q, want the usage message", args, stderr)
		}
	}
}

func TestRun_InvalidThresholds(t *testing.T) {
	t.Parallel()

	for _, threshold := range []string{"abc", "-1", "100.5", ""} {
		code, _, stderr := runWithReport(t, []string{threshold}, sampleReport)

		if code != exitUsage {
			t.Errorf("threshold %q: exit code = %d, want %d", threshold, code, exitUsage)
		}

		if !strings.Contains(stderr, "invalid threshold") {
			t.Errorf("threshold %q: stderr = %q, want the invalid-threshold message", threshold, stderr)
		}
	}
}

func TestParseTotalPercent(t *testing.T) {
	t.Parallel()

	pct, err := parseTotalPercent("total:\t\t\t\t(statements)\t96.3%")
	if err != nil {
		t.Fatalf("parseTotalPercent() error = %v", err)
	}

	if pct != 96.3 {
		t.Errorf("pct = %v, want 96.3", pct)
	}
}

func TestParseTotalPercent_TrailingGarbage(t *testing.T) {
	t.Parallel()

	if _, err := parseTotalPercent("total: (statements) 96.3 percent"); err == nil {
		t.Error("parseTotalPercent() error = nil, want an error for non-percent trailing text")
	}
}
