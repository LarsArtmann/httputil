#!/usr/bin/env bash
# Pre-commit hook: runs golangci-lint on staged Go files.
# Fails the commit if any lint issues are found.
set -euo pipefail

STAGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -z "$STAGED_GO_FILES" ]; then
	exit 0
fi

echo "Running golangci-lint on staged Go files..."

if ! golangci-lint run --timeout=5m; then
	echo ""
	echo "::error::golangci-lint found issues. Fix them before committing."
	echo "Run 'golangci-lint run --fix' to auto-fix what's possible."
	exit 1
fi

echo "golangci-lint: clean."
