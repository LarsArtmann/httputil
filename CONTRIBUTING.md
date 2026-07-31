# Contributing to httputil

Thank you for your interest in contributing to httputil.

## Development Setup

```bash
go test ./...              # Run tests
go test -race ./...        # Race detection
go vet ./...               # Vet
golangci-lint run          # Lint (~70 linters)
golangci-lint run --fix    # Auto-fix what's possible
golangci-lint fmt          # Format (gofumpt + golines@120 + gci)
govulncheck ./...          # Vulnerability scan
```

`golangci-lint run` is the authoritative quality gate. Run `golangci-lint fmt` after editing — manual whitespace will likely violate `wsl_v5`.

### Nix Flake Apps

The `flake.nix` provides task automation apps. Use these for canonical entry points:

| App                  | Command               | Purpose                                 |
| -------------------- | --------------------- | --------------------------------------- |
| `nix run .#test`     | `go test -race`       | Run tests with race detection           |
| `nix run .#build`    | `go build ./...`      | Build all packages                      |
| `nix run .#vet`      | `go vet ./...`        | Run go vet                              |
| `nix run .#lint`     | `golangci-lint run`   | Run the full linter suite               |
| `nix run .#coverage` | `go test -cover`      | Generate coverage report                |
| `nix run .#clean`    | `go clean -testcache` | Clean test cache and coverage artifacts |

### Minimum Go Version

Go **1.26.0+**. The `go.mod` toolchain directive may pin a specific patch release (currently 1.26.5), but the minimum supported version is the `.0` patch of the declared minor.

## Code Style

- Follow `func(http.Handler) http.Handler` middleware signature
- Allowed dependencies: `$gostd`, `go-error-family`, `golang.org/x/time`, and `justinas/nosurf` only (enforced by `depguard`)
- Every struct field must be set (`exhaustruct` linter)
- Package-level sentinel errors only (no inline `errors.New`)
- Comments end with periods (`godot`)
- Named constants for magic numbers (`mnd`)
- All tests call `t.Parallel()` first (`paralleltest`)
- No inline error checks: use separate assignment then check (`noinlineerr`)
- Header keys in canonical MIME form (`canonicalheader`)

See `AGENTS.md` for the full list of non-obvious lint constraints.

## Versioning Policy

httputil follows [Semantic Versioning](https://semver.org/) with the 0.x convention:

- **Pre-1.0 (current):** Breaking changes are acceptable in minor versions. Consumers pinning a specific version are not affected; consumers using `@latest` or minor ranges should review CHANGELOG entries marked **Breaking:** before upgrading.
- **1.0+:** Breaking changes require a major version bump. The frozen API surface will be documented in `docs/v1-stability.md`.

## CHANGELOG

All notable changes are documented in [CHANGELOG.md](CHANGELOG.md) following [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) format.

When opening a PR:

- Add an entry under `[Unreleased]` with the appropriate section: `Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`, or `Security`.
- Mark breaking changes with a **Breaking:** prefix.
- Reference the PR or issue number when relevant.

## Pull Requests

- Ensure `go test -race ./...` passes
- Ensure `golangci-lint run` reports zero issues
- Ensure `govulncheck ./...` reports no vulnerabilities
- Add tests for new functionality
- Update documentation (README, CHANGELOG) as needed

## Release Process

See [docs/RELEASE.md](docs/RELEASE.md) for the full release runbook.

## License

By contributing, you agree that your contributions will be licensed under the same terms as the project (see LICENSE).
