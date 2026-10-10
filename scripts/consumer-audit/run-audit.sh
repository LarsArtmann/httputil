#!/usr/bin/env bash
# Consumer-audit pipeline for github.com/larsartmann/httputil.
# Automates steps 1-5 of the 2026-10-08 audit method (docs/review/lint/000-index.md);
# step 6 (per-pattern human review) stays manual.
set -euo pipefail

OUT="${1:-/tmp/httputil-consumer-audit-$(date +%Y-%m-%d)}"
PROJECTS_DIR="${PROJECTS_DIR:-$HOME/projects}"
MODULE="github.com/larsartmann/httputil"

mkdir -p "$OUT"

echo "== Step 1: who-uses inventory -> $OUT/who-uses.txt"
if [ -d "$PROJECTS_DIR/project-dependency-graph" ]; then
	(cd "$PROJECTS_DIR/project-dependency-graph" && GOTOOLCHAIN=auto go run . who-uses "$MODULE" --dir "$PROJECTS_DIR" --direct-only=false) \
		>"$OUT/who-uses.txt"
	wc -l <"$OUT/who-uses.txt" | xargs echo "consumers listed:"
else
	echo "SKIP: $PROJECTS_DIR/project-dependency-graph not found" >&2
fi

echo "== Step 2: per-consumer go.mod pins -> $OUT/pins.txt"
(cd "$PROJECTS_DIR" && rg -n -g 'go.mod' -g '!**/vendor/**' "$MODULE" .) >"$OUT/pins.txt" || true
echo "pins recorded: $(wc -l <"$OUT/pins.txt")"

echo "== Step 3: call-site corpus -> $OUT/corpus.txt"
(cd "$PROJECTS_DIR" && rg -n -g '*.go' -g '!**/vendor/**' -g '!**/testdata/**' 'httputil\.' .) >"$OUT/corpus.txt" || true
echo "call sites: $(wc -l <"$OUT/corpus.txt")"

echo "== Step 4: pattern-pack greps -> $OUT/patterns/"
mkdir -p "$OUT/patterns"
while IFS=$'\t' read -r name pattern; do
	[ -z "$name" ] && continue
	case "$name" in \#*) continue ;; esac
	(cd "$PROJECTS_DIR" && rg -n -g '*.go' -g '!**/vendor/**' -g '!**/testdata/**' -e "$pattern" </dev/null) \
		>"$OUT/patterns/$name.txt" || true
	echo "  $name: $(wc -l <"$OUT/patterns/$name.txt") hits"
done <"$(dirname "$0")/patterns.tsv"

echo "== Step 5: report skeleton -> $OUT/report.md"
sed "s/DATE/$(date +%Y-%m-%d)/; s|OUTDIR|$OUT|g" "$(dirname "$0")/report-template.md" >"$OUT/report.md"

echo "Done. Step 6 (human per-pattern review) starts at $OUT/report.md"
