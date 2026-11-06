#!/usr/bin/env bash
# prerelease-check.sh — automate the pre-release gates documented in
# docs/RELEASE.md (Pre-Release Verification). Run this before cutting a version tag; every gate must pass.
#
# Usage: scripts/prerelease-check.sh [--skip-flake]
#   --skip-flake  skip the nix flake check (gate 9); CI parity for runners
#                 without nix — the flake gates must still run somewhere.
set -uo pipefail

cd "$(dirname "$0")/.." || exit 1

skip_flake=false
for arg in "$@"; do
	case "$arg" in
	--skip-flake) skip_flake=true ;;
	*)
		printf 'unknown argument: %s\nusage: prerelease-check.sh [--skip-flake]\n' "$arg" >&2
		exit 2
		;;
	esac
done

step() { printf '\n=== %s ===\n' "$1"; }
fail() {
	printf '\n!!! FAILED: %s\n' "$1" >&2
	exit 1
}

step "1/9 Working tree must be clean"
[[ -z "$(git status --porcelain)" ]] || fail "working tree is dirty; commit or stash first"
echo "clean"

step "2/9 Build (workspace)"
go build ./... || fail "go build ./..."

step "3/9 Vet (root + server_timing + endpoints)"
go vet ./... || fail "go vet ./... (root)"
(cd server_timing && go vet ./...) || fail "go vet ./... (server_timing)"
(cd endpoints && go vet ./...) || fail "go vet ./... (endpoints)"

step "3b/9 API compatibility vs released baselines (apidiff)"
# Single source of the gate logic (the same script ci.yml runs); baselines
# and the regeneration ritual live in apidiff/README.md, the promise itself
# in docs/v1-stability.md.
./scripts/check-apidiff.sh apidiff/httputil.export.zip . "root module" "v1.7.0" || fail "apidiff (root module)"
./scripts/check-apidiff.sh apidiff/server_timing.export.zip server_timing "server_timing module" "v1.0.3" || fail "apidiff (server_timing)"
./scripts/check-apidiff.sh apidiff/endpoints.export.zip endpoints "endpoints module" "endpoints/v0.1.0" || fail "apidiff (endpoints)"

step "4/9 Tests with race detection (root + server_timing + endpoints)"
go test -race -count=1 ./... || fail "go test -race (root)"
(cd server_timing && go test -race -count=1 ./...) || fail "go test -race (server_timing)"
(cd endpoints && go test -race -count=1 ./...) || fail "go test -race (endpoints)"

step "4a/9 Error-sentinel guards (root + server_timing)"
# Named explicitly so a rename or deletion of the guard cannot silently drop
# the gate: these are the tests that catch the 2026-10-15 sweep class. The
# server_timing module has no sentinels yet; its guard is the backstop for when
# it adopts classified errors. The endpoints module carries plain stdlib
# sentinels today.
go test -race -count=1 -run 'Sentinel' . || fail "sentinel guard (root)"
(cd server_timing && go test -race -count=1 -run 'Sentinel' .) || fail "sentinel guard (server_timing)"
(cd endpoints && go test -race -count=1 -run 'Sentinel' .) || fail "sentinel guard (endpoints)"

step "4b/9 Race stress (CI parity, -count=10)"
# count=1 does not reliably surface timing-dependent races (the 2026-08-05
# lesson: clean at count=1, failing 60% of -count=10 runs); CI runs this on
# every push, so release parity is the point.
go test -race -count=10 ./... || fail "go test -race -count=10 (root)"

step "5/9 Lint (root + server_timing + endpoints)"
golangci-lint run ./... || fail "golangci-lint run (root)"
(cd server_timing && golangci-lint run ./...) || fail "golangci-lint run (server_timing)"
(cd endpoints && golangci-lint run ./...) || fail "golangci-lint run (endpoints)"

step "5b/9 Doc snippet symbol check (CI parity)"
# Same command as ci.yml's "Doc snippet symbol check": README and the
# docs/integrations snippets must reference symbols that still exist.
go run ./scripts/doc-snippet-refs README.md docs/integrations/*.md ||
	fail "doc-snippet-refs"

step "6/9 erraudit gates"
erraudit lint ./... --type legacy_as || fail "erraudit legacy_as"
erraudit lint ./... --type stdlib_constructor --enforce-go-error-family ||
	fail "erraudit stdlib_constructor"

step "6b/9 erraudit gates (nested modules; lint is root-only)"
# Per-root because nested-module handling is build-dependent: erraudit fa5575f
# (deployed until the 2026-11-03 fixes, erraudit#12/#13) analyzes the root
# module only and rejects --modules on lint; HEAD builds analyze all modules
# by default. Per-root gating is correct on both (owner ruling 2026-11-06:
# release gate, not CI).
(cd endpoints && erraudit lint ./... --type legacy_as) ||
	fail "erraudit legacy_as (endpoints)"
(cd endpoints && erraudit lint ./... --type stdlib_constructor --enforce-go-error-family) ||
	fail "erraudit stdlib_constructor (endpoints)"
(cd server_timing && erraudit lint ./... --type legacy_as) ||
	fail "erraudit legacy_as (server_timing)"
(cd server_timing && erraudit lint ./... --type stdlib_constructor --enforce-go-error-family) ||
	fail "erraudit stdlib_constructor (server_timing)"

step "7/9 Coverage threshold (95%)"
# Library coverage only: the scripts/ dev tooling (doc-snippet-refs has no
# tests by design; coverage-threshold tests itself) would drag the total
# below the gate.
mapfile -t coverage_packages < <(go list ./... | grep -v /scripts/)
go test "${coverage_packages[@]}" -coverprofile=coverage.out || fail "go test -coverprofile"
go tool cover -func=coverage.out | go run ./scripts/coverage-threshold 95 || fail "coverage below 95%"

step "8/9 CHANGELOG has an [Unreleased] section"
grep -q "## \[Unreleased\]" CHANGELOG.md || fail "CHANGELOG.md has no [Unreleased] section"
echo "found"

step "8b/9 CHANGELOG link definitions (CI parity)"
./scripts/check-changelog-links.sh || fail "check-changelog-links.sh"

step "8c/9 Q-value corpus sync (shared parser parity oracle)"
./scripts/check-qvalue-corpus-sync.sh || fail "check-qvalue-corpus-sync.sh"

step "9/9 Flake gates"
if [[ "$skip_flake" == true ]]; then
	echo "skipped (--skip-flake)"
else
	nix flake check || fail "nix flake check"
fi

printf '\nAll pre-release gates passed. Proceed with the docs/RELEASE.md release-time steps (7. CHANGELOG ... 16. cleanup).\n'
