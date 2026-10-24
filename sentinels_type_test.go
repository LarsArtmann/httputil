package httputil

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSentinelsKeepConcreteType guards the 2026-10-15 incident class: an
// automated sweep annotated error-family sentinels as plain error, which
// strips WithContext/WithCause from the sentinel and breaks every clone call
// site that uses them directly. Sentinels must keep their concrete
// *errorfamily.Error type by omitting the annotation (AGENTS.md, erraudit
// block).
func TestSentinelsKeepConcreteType(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}

	fset := token.NewFileSet()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, entry.Name(), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}

		assertNoPlainErrorSentinels(t, fset, file)
	}
}

// TestSentinelGuardDetectsPlainErrorAnnotation proves the guard is not vacuous:
// a file declaring an err* variable with the plain error type must be reported.
// Without this, a broken detector and a clean package would look identical.
func TestSentinelGuardDetectsPlainErrorAnnotation(t *testing.T) {
	t.Parallel()

	const source = `package fixture

var errSwept error
`

	path := filepath.Join(t.TempDir(), "fixture.go")

	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	report := new(recordingReporter)
	assertNoPlainErrorSentinels(report, fset, file)

	if len(report.findings) != 1 {
		t.Fatalf("guard reported %d findings for one annotated sentinel, want 1", len(report.findings))
	}

	if !strings.Contains(report.findings[0], "errSwept") {
		t.Errorf("guard finding %q does not name errSwept", report.findings[0])
	}
}

// sentinelReporter is the part of testing.T the guard needs, so the detector can
// be exercised by a fake reporter.
type sentinelReporter interface {
	Helper()
	Errorf(format string, args ...any)
}

// recordingReporter collects guard findings for the self-test.
type recordingReporter struct {
	findings []string
}

// Helper marks the recorder as a test helper.
func (*recordingReporter) Helper() {}

// Errorf records a formatted finding.
func (r *recordingReporter) Errorf(format string, args ...any) {
	r.findings = append(r.findings, fmt.Sprintf(format, args...))
}

// assertNoPlainErrorSentinels reports every err*/Err* variable declared with
// an explicit error type in file.
func assertNoPlainErrorSentinels(t sentinelReporter, fset *token.FileSet, file *ast.File) {
	t.Helper()

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || valueSpec.Type == nil {
				continue
			}

			typeIdent, ok := valueSpec.Type.(*ast.Ident)
			if !ok || typeIdent.Name != "error" {
				continue
			}

			reportPlainErrorSentinel(t, fset, valueSpec)
		}
	}
}

// reportPlainErrorSentinel reports the err*/Err* names in spec that carry the
// plain error type.
func reportPlainErrorSentinel(t sentinelReporter, fset *token.FileSet, spec *ast.ValueSpec) {
	t.Helper()

	for _, name := range spec.Names {
		if !strings.HasPrefix(name.Name, "err") && !strings.HasPrefix(name.Name, "Err") {
			continue
		}

		t.Errorf(
			"%s: sentinel %s declared as plain error; keep the concrete error-family type so WithContext/WithCause call sites compile",
			fset.Position(name.Pos()),
			name.Name,
		)
	}
}
