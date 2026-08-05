#!/usr/bin/env bash
# Updates the coverage badge in README.md to reflect the current test
# coverage percentage. Reads coverage.out (produced by `go test -coverprofile`)
# and rewrites the badge URL in-place.
#
# Usage: update-coverage-badge.sh <coverage.out> <README.md>
set -euo pipefail

COVERAGE_FILE="${1:-coverage.out}"
README="${2:-README.md}"

if [ ! -f "$COVERAGE_FILE" ]; then
    echo "ERROR: coverage file $COVERAGE_FILE not found"
    exit 1
fi

if [ ! -f "$README" ]; then
    echo "ERROR: README file $README not found"
    exit 1
fi

# Compute total coverage percentage from coverage.out.
# go tool cover -func prints "<name>  <pct>%  <count>"; the line starting
# with "total:" holds the aggregate.
total=$(go tool cover -func="$COVERAGE_FILE" | awk '/^total:/ { gsub("%", "", $3); print $3 }')

if [ -z "$total" ]; then
    echo "ERROR: could not extract coverage from $COVERAGE_FILE"
    exit 1
fi

# Round to one decimal place for stable display.
formatted=$(printf "%.1f" "$total")

# Pick a color: red < 70, yellow 70-89, green >= 90.
# Aligns with shields.io conventions for coverage badges.
color=$(awk -v t="$total" 'BEGIN { if (t >= 90) print "green"; else if (t >= 70) print "yellow"; else print "red" }')

# Replace the badge line in the README. The pattern matches the static
# 97.8% green badge and any prior dynamic value.
new_badge="[![Coverage](https://img.shields.io/badge/coverage-${formatted}%25-${color})](#)"

if grep -q 'shields.io/badge/coverage-' "$README"; then
    sed -i.bak "s|!\\[Coverage\\](https://img.shields.io/badge/coverage-[^)]*)|${new_badge}|g" "$README"
    rm -f "$README.bak"
    echo "Updated coverage badge: ${formatted}% (${color})"
else
    echo "WARNING: no coverage badge found in $README — skipping update"
fi