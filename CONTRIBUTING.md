# Contributing to httputil

Thank you for your interest in contributing to httputil.

## Development Setup

```bash
go test ./...              # Run tests
golangci-lint run          # Lint (~70 linters)
golangci-lint run --fix    # Auto-fix what's possible
golangci-lint fmt          # Format (gofumpt + golines@120 + gci)
```

`golangci-lint run` is the authoritative quality gate.

## Code Style

- Follow `func(http.Handler) http.Handler` middleware signature
- Allowed dependencies: `$gostd`, `go-error-family`, and `golang.org/x/time` only (enforced by `depguard`)
- Every struct field must be set (`exhaustruct` linter)
- Package-level sentinel errors only (no inline `errors.New`)
- Comments end with periods (`godot`)
- Named constants for magic numbers (`mnd`)
- All tests call `t.Parallel()` first

## Pull Requests

- Ensure `go test ./...` passes
- Ensure `golangci-lint run` reports zero issues
- Add tests for new functionality
- Update documentation (README, CHANGELOG) as needed

## License

By contributing, you agree that your contributions will be licensed under the same terms as the project (see LICENSE).
