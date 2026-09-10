#!/usr/bin/env bash
# prerelease-check.sh — automate the pre-release gates documented in
# RELEASE.md. Run this before cutting a version tag; every gate must pass.
set -uo pipefail

cd "$(dirname "$0")/.."

# /mnt/buildcache is unwritable in some environments; without these exports
# every Go toolchain call fails with "failed to initialize build cache".
export GOCACHE="${GOCACHE:-$HOME/.cache/go-build-httputil}"
export GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-$HOME/.cache/golangci-lint-httputil}"
export GOEXPERIMENT=jsonv2

step() { printf '\n=== %s ===\n' "$1"; }
fail() { printf '\n!!! FAILED: %s\n' "$1" >&2; exit 1; }

step "1/9 Working tree must be clean"
[[ -z "$(git status --porcelain)" ]] || fail "working tree is dirty; commit or stash first"
echo "clean"

step "2/9 Build (workspace)"
go build ./... || fail "go build ./..."

step "3/9 Vet (root + server_timing)"
go vet ./... || fail "go vet ./... (root)"
(cd server_timing && go vet ./...) || fail "go vet ./... (server_timing)"

step "4/9 Tests with race detection (root + server_timing)"
go test -race -count=1 ./... || fail "go test -race (root)"
(cd server_timing && go test -race -count=1 ./...) || fail "go test -race (server_timing)"

step "5/9 Lint (root + server_timing)"
golangci-lint run ./... || fail "golangci-lint run (root)"
(cd server_timing && golangci-lint run ./...) || fail "golangci-lint run (server_timing)"

step "6/9 erraudit gates"
GOEXPERIMENT=jsonv2 erraudit lint ./... --type legacy_as || fail "erraudit legacy_as"
GOEXPERIMENT=jsonv2 erraudit lint ./... --type stdlib_constructor --enforce-go-error-family \
	|| fail "erraudit stdlib_constructor"

step "7/9 Coverage threshold (95%)"
go test -coverprofile=coverage.out ./... || fail "go test -coverprofile"
go tool cover -func=coverage.out | go run ./scripts/coverage-threshold 95 || fail "coverage below 95%"

step "8/9 CHANGELOG has an [Unreleased] section"
grep -q "## \[Unreleased\]" CHANGELOG.md || fail "CHANGELOG.md has no [Unreleased] section"
echo "found"

step "9/9 Flake gates"
nix flake check || fail "nix flake check"

printf '\nAll pre-release gates passed. Proceed with the RELEASE.md tag checklist.\n'
