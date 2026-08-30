# Release Runbook

Step-by-step checklist for cutting an httputil release. Follow in order — each phase gates the next.

## Pre-Release Verification

Run every gate below. If any fails, fix it before proceeding.

### 1. Clean build

```bash
go build ./...
go vet ./...
```

### 2. Full test suite with race detection

```bash
go test -race -count=1 ./...
```

### 3. Coverage measurement

```bash
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1
```

Record the total percentage. If it dropped below 95%, investigate before releasing (the documented gate threshold is 95%).

### 4. govulncheck

```bash
govulncheck ./...
```

Must report "No vulnerabilities found." If vulnerabilities are found in a direct dependency, patch or document before releasing.

### 5. Lint gate

```bash
golangci-lint run --max-issues-per-linter=0 --max-same-issues=0
golangci-lint fmt
```

Must report zero issues.

### 6. Benchmarks (regression check)

```bash
go test -bench=. -benchmem -count=1 -run=^$ ./...
```

Compare against the previous release. Investigate any regression exceeding 10%.

Then refresh the recorded baseline in `docs/benchmarks.md` (3s × 5 protocol) so the doc reflects the release being cut. Note provenance for any row measured with a different harness or protocol than the rest.

## Release-Time Steps

### 7. Update CHANGELOG.md

Move entries from `[Unreleased]` to a new `[X.Y.Z] - YYYY-MM-DD` section. Follow [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format.

Mark breaking changes with **Breaking:** prefix.

### 8. Update FEATURES.md

Ensure the feature inventory reflects the current state. Move completed items from PLANNED to DONE.

### 9. Annotate historical reports

Search for open claims that this release resolves:

```bash
rg -l 'GOEXPERIMENT|json/v2|biggest open risk|open question' docs/
```

Append a resolution note to each matching report (see the `update-old-docs` skill convention: non-destructive inline correction or end-of-file appendix).

### 10. Commit all changes

```bash
git add -A
git commit -m "release: vX.Y.Z"
```

### 11. Tag (SSH-signed annotated)

```bash
git tag -a -s vX.Y.Z -m "Release vX.Y.Z"
```

Verify the signature:

```bash
git tag -v vX.Y.Z
```

### 12. Push

```bash
git push origin master
git push origin vX.Y.Z
```

### 13. Create GitHub Release

Extract the CHANGELOG section for this version as release notes:

```bash
gh release create vX.Y.Z --notes "$(awk '/^## \[X.Y.Z\]/{f=1} f{print} /^## \[/{if(f&&!first){exit}; first=1}' CHANGELOG.md)"
```

Or write custom notes for major releases.

### 14. Verify the release

```bash
gh release view vX.Y.Z
```

Confirm the tag, release notes, and source tarball appear on the GitHub releases page.

## Post-Release

### 15. Consumer verification

In a clean environment:

```bash
go get github.com/larsartmann/httputil@vX.Y.Z
go mod verify
govulncheck ./...
```

### 16. Clean up

```bash
trash-put coverage.out  # or add to .gitignore
```

## Versioning Policy

- **0.x**: Breaking changes are acceptable in minor versions. Semver 0.x convention.
- **1.0+**: Breaking changes require a major version bump. The frozen API surface is documented in `docs/v1-stability.md`.

## Superseded Releases

If a prior tag on origin is broken for consumers, note it in the GitHub Release for the fix:

> **Note:** vX.Y.Z-1 contained a build-breaking issue. Upgrade to vX.Y.Z.

Do not delete or re-create historical tags — they are immutable history.

## Architecture Diagram Regeneration

The D2 diagrams in `docs/architecture-understanding/` are rendered with the
`d2` binary pinned in `flake.nix` (devShell package, version follows the
locked nixpkgs input — currently d2 v0.7.1 with the elk layout engine).
Regenerate after editing a `.d2` source:

```bash
nix develop -c d2 --layout elk docs/architecture-understanding/<file>.d2 docs/architecture-understanding/<file>.svg
```

Do not render with an ad-hoc d2 install — different versions produce
different SVG layouts, which makes historical diffs noisy.
