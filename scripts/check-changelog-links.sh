#!/usr/bin/env bash
# Validates CHANGELOG.md: every [version] heading must have a matching link
# definition, and vice versa. Also verifies the [Unreleased] link format.
set -euo pipefail

CHANGELOG="${1:-CHANGELOG.md}"

headings=$(sed -n 's/^## \[\([^]]*\)\].*/\1/p' "$CHANGELOG" | sort)
links=$(sed -n 's/^\[\([^]]*\)\]:.*/\1/p' "$CHANGELOG" | sort)

missing_links=$(comm -23 <(echo "$headings") <(echo "$links"))
if [ -n "$missing_links" ]; then
	echo "ERROR: CHANGELOG headings without link definitions:"
	echo "$missing_links"
	exit 1
fi

extra_links=$(comm -13 <(echo "$headings") <(echo "$links"))
if [ -n "$extra_links" ]; then
	echo "ERROR: CHANGELOG link definitions without headings:"
	echo "$extra_links"
	exit 1
fi

if ! grep -q '^\[Unreleased\]:.*\.\.\.HEAD$' "$CHANGELOG"; then
	echo "ERROR: [Unreleased] link must end with '...HEAD'"
	exit 1
fi

echo "CHANGELOG links are consistent."
