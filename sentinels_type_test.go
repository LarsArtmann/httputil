package httputil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
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

// assertNoPlainErrorSentinels reports every err*/Err* variable declared with
// an explicit error type in file.
func assertNoPlainErrorSentinels(t *testing.T, fset *token.FileSet, file *ast.File) {
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
func reportPlainErrorSentinel(t *testing.T, fset *token.FileSet, spec *ast.ValueSpec) {
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
