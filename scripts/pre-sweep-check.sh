#!/usr/bin/env bash
# pre-sweep-check.sh — the local stopgap for the auto-commit daemon's missing
# pre-sweep build gate (the 2026-10-14 red-master class: the daemon commits and
# pushes non-compiling intermediates, so a broken tree reaches every consumer
# before anyone runs the full suite).
#
# Run this before leaving a tree unattended (end of session, before a long task,
# before handing the worktree to another agent). It answers one question: would
# the daemon commit something that does not compile?
#
# Usage: scripts/pre-sweep-check.sh
#
# Scope is deliberately narrow and fast (build + vet + the sentinel guards, both
# modules). Full gates stay in scripts/prerelease-check.sh; tests, lint, and the
# OTEL interop fixture are not repeated here.
set -uo pipefail

cd "$(dirname "$0")/.." || exit 1

# /mnt/buildcache is unwritable in some environments; without these exports
# every Go toolchain call fails with "failed to initialize build cache".
export GOCACHE="${GOCACHE:-$HOME/.cache/go-build-httputil}"
export GOLANGCI_LINT_CACHE="${GOLANGCI_LINT_CACHE:-$HOME/.cache/golangci-lint-httputil}"
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

step() { printf '\n=== %s ===\n' "$1"; }
fail() {
	printf '\n!!! BROKEN TREE: %s\n' "$1" >&2
	printf 'Do not leave this worktree unattended; the daemon would commit it.\n' >&2
	exit 1
}

step "1/3 Build (root + server_timing)"
go build ./... || fail "go build ./... (root)"
(cd server_timing && go build ./...) || fail "go build ./... (server_timing)"

step "2/3 Vet (root + server_timing)"
go vet ./... || fail "go vet ./... (root)"
(cd server_timing && go vet ./...) || fail "go vet ./... (server_timing)"

step "3/3 Error-sentinel guards (root + server_timing)"
go test -count=1 -run 'Sentinel' . || fail "sentinel guard (root)"
(cd server_timing && go test -count=1 -run 'Sentinel' .) || fail "sentinel guard (server_timing)"

printf '\nOK: the tree compiles, vets clean, and keeps its sentinels concrete.\n'
