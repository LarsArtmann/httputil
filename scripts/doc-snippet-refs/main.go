// doc-snippet-refs checks that Go code fences in markdown docs only reference
// exported symbols that actually exist in this workspace's packages. Markdown
// snippets are never compiled, so a renamed or removed API (the Recovery(nil)
// incident) used to ship silently; this checker makes such drift a build
// failure. Fences that do not parse as Go — bare-statement fragments — are
// skipped: the tool targets symbol drift, not fragment compilability.
//
// Usage: go run ./scripts/doc-snippet-refs docs/integrations/*.md README.md
package main

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"os"
	"regexp"
	"sort"
)

// checkedPackages maps the import alias used inside snippets to the package
// whose exported API the selector must resolve against.
//
//nolint:gochecknoglobals // static configuration table, never mutated
var checkedPackages = map[string]string{
	"httputil":     "github.com/larsartmann/httputil",
	"servertiming": "github.com/larsartmann/httputil/server_timing",
	"httpspec":     "github.com/larsartmann/httputil/httpspec",
}

var fencePattern = regexp.MustCompile("(?s)```go\n(.*?)```")

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: doc-snippet-refs <markdown files...>")

		os.Exit(2)
	}

	apis := make(map[string]map[string]bool, len(checkedPackages))
	for alias, pkgPath := range checkedPackages {
		names, err := exportedNames(pkgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "doc-snippet-refs: loading %s: %v\n", pkgPath, err)

			os.Exit(2)
		}

		apis[alias] = names
	}

	findings := 0

	for _, path := range os.Args[1:] {
		content, err := os.ReadFile(path) //nolint:gosec // paths come from the invoker
		if err != nil {
			fmt.Fprintf(os.Stderr, "doc-snippet-refs: %v\n", err)

			os.Exit(2)
		}

		for _, match := range fencePattern.FindAllSubmatch(content, -1) {
			refs := unknownSelectors(apis, string(match[1]))

			for _, alias := range sortedAliases(refs) {
				for _, name := range refs[alias] {
					fmt.Fprintf(os.Stderr, "%s: no exported symbol %s.%s\n", path, alias, name)

					findings++
				}
			}
		}
	}

	if findings > 0 {
		fmt.Fprintf(os.Stderr, "doc-snippet-refs: %d unknown symbol reference(s)\n", findings)

		os.Exit(1)
	}

	fmt.Fprintln(os.Stdout, "doc-snippet-refs: all snippet references resolve")
}

// exportedNames returns the exported package-level names of pkgPath, resolved
// from source so the check sees the real API rather than a copy-pasted list.
func exportedNames(pkgPath string) (map[string]bool, error) {
	fset := token.NewFileSet()

	pkg, err := importer.ForCompiler(fset, "source", nil).Import(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("import: %w", err)
	}

	names := make(map[string]bool, len(pkg.Scope().Names()))

	for _, name := range pkg.Scope().Names() {
		if ast.IsExported(name) {
			names[name] = true
		}
	}

	return names, nil
}

// unknownSelectors parses one code fence and returns every pkg.Sel reference
// whose Sel is not an exported symbol of the referenced workspace package.
// A fence is parsed as a standalone file first; if that fails, as a bare
// fragment wrapped in a function; if both fail, it is skipped.
func unknownSelectors(apis map[string]map[string]bool, fence string) map[string][]string {
	unknown := make(map[string][]string)

	node := parseFence(fence)
	if node == nil {
		return unknown
	}

	ast.Inspect(node, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		if _, checked := checkedPackages[ident.Name]; !checked {
			return true
		}

		name := sel.Sel.Name

		if !ast.IsExported(name) {
			return true
		}

		if !apis[ident.Name][name] {
			unknown[ident.Name] = append(unknown[ident.Name], name)
		}

		return true
	})

	for alias := range unknown {
		sort.Strings(unknown[alias])
	}

	return unknown
}

// parseFence parses the fence as a standalone file, falling back to wrapping
// it in a function for bare-statement fragments, returning nil when neither
// parses.
func parseFence(fence string) ast.Node {
	fset := token.NewFileSet()

	if node, err := parser.ParseFile(fset, "snippet.go", fence, 0); err == nil {
		return node
	}

	wrapped := "package snippet\n\nfunc _() {\n" + fence + "\n}\n"

	if node, err := parser.ParseFile(fset, "fragment.go", wrapped, 0); err == nil {
		return node
	}

	return nil
}

func sortedAliases(refs map[string][]string) []string {
	aliases := make([]string, 0, len(refs))

	for alias := range refs {
		aliases = append(aliases, alias)
	}

	sort.Strings(aliases)

	return aliases
}
