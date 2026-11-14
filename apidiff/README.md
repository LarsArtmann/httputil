# API Compatibility Baselines

Machine-checked floor of the frozen v1 API promise documented in
[docs/v1-stability.md](../docs/v1-stability.md). Each zip holds the export data
of every exported package of one module at the latest released tag.

| File                       | Module                                          | Generated at               |
| -------------------------- | ----------------------------------------------- | -------------------------- |
| `httputil.export.zip`      | `github.com/larsartmann/httputil`               | tag `v1.9.0`               |
| `server_timing.export.zip` | `github.com/larsartmann/httputil/server_timing` | tag `v1.9.0` (coordinated) |
| `endpoints.export.zip`     | `github.com/larsartmann/httputil/endpoints`     | tag `v1.9.0` (coordinated) |

## The gate

From each module directory (`GOWORK=off`), an incompatible-change check must
print nothing:

```bash
APIDIFF=golang.org/x/exp/cmd/apidiff@v0.0.0-20261014143641-f0251eb4aa3e
GOWORK=off go run $APIDIFF -m -incompatible apidiff/httputil.export.zip .      # repo root
GOWORK=off go run $APIDIFF -m -incompatible apidiff/server_timing.export.zip . # server_timing/
GOWORK=off go run $APIDIFF -m -incompatible apidiff/endpoints.export.zip . # endpoints/
```

`apidiff` exits 0 even when it reports incompatible changes (verified
2026-11-02), so the gate fails on non-empty stdout, not on exit code.
`docs/v1-stability.md` and `scripts/prerelease-check.sh` are the canonical
users of this contract.

## Regeneration (release time only)

After a release passes all gates, regenerate both zips from the release
commit and land them in the same release push, so the floor always equals
the newest tag:

```bash
git worktree add /tmp/apidiff-root v1.9.0   # the new tag
cd /tmp/apidiff-root && GOWORK=off go run $APIDIFF -m -w <repo>/apidiff/httputil.export.zip .
cd /tmp/apidiff-root/server_timing && GOWORK=off go run $APIDIFF -m -w <repo>/apidiff/server_timing.export.zip .
cd /tmp/apidiff-root/endpoints && GOWORK=off go run $APIDIFF -m -w <repo>/apidiff/endpoints.export.zip .
git worktree remove /tmp/apidiff-root
```

Export data is written by the repo's Go toolchain; if a toolchain bump makes
an old baseline unreadable (read errors in the gate), regenerate from the
stored tag and commit — that is a format refresh, not a compatibility change.
