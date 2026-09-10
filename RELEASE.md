# Release Process

How to cut a version tag for `httputil` and `httputil/server_timing`. The mechanical gates are automated in `scripts/prerelease-check.sh`; this page is the checklist around them.

## 1. Pre-release gates (automated)

Run from the repository root:

```bash
./scripts/prerelease-check.sh
```

The script enforces, in order: clean working tree, workspace build, `go vet` for both modules, `go test -race` for both modules, `golangci-lint run` for both modules, both erraudit gates (no legacy `errors.As`, no inline stdlib error constructors), the 95% coverage threshold via `scripts/coverage-threshold`, a `[Unreleased]` section in `CHANGELOG.md`, and `nix flake check`.

## 2. Finalize the changelog

1. Rename the `## [Unreleased]` section to `## [vX.Y.Z] — YYYY-MM-DD`.
2. Start a fresh empty `## [Unreleased]` section above it.
3. The released section is now **frozen** (see AGENTS.md, CHANGELOG Freeze Policy): corrections to already-released work go into the new `[Unreleased]`, never into the frozen section.

## 3. Tag

```bash
git tag -a vX.Y.Z -m "vX.Y.Z: <one-line summary>"
```

Do not re-tag a version that was already pushed: the Go module proxy caches tags permanently (see the `go-release` runbook for recovery options).

## 4. Push

```bash
git push origin master && git push origin vX.Y.Z
```

The pushed `v*` tag triggers `.github/workflows/release.yml`: build, vet (root + `server_timing`), race tests (root + `server_timing`), lint, govulncheck (both modules), then a GitHub Release with generated notes.

## 5. Post-release verification

1. `https://pkg.go.dev/github.com/larsartmann/httputil@vX.Y.Z` resolves (may lag a few minutes).
2. `GOFLAGS=-mod=mod go get github.com/larsartmann/httputil@vX.Y.Z` works from a scratch module.
3. Update the coverage badge/README references if the release changed public APIs.

## Versioning notes

- Both modules release together under one tag; `server_timing` is resolved through the `replace` directive and `go.work`, so consumers of the root module never see a version skew.
- Pre-release tags (`vX.Y.Z-rc.N`) are marked prerelease by the release workflow automatically.
