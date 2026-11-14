# Fleet golangci v2.15 wrapcheck-wave matrix — 2026-11-14

Read-only scan of every `~/projects` repo importing `github.com/larsartmann/go-error-family`
(~250 go.mod files, ~150 repos), recording CI lint pins, GOTOOLCHAIN pins, clone call sites
(non-test `.WithContext*`/`.WithCause`), and the action taken in session 3. Regenerate with the
scan loop in docs/status/2026-11-14_11-27 §f.7 before acting; this file is the 2026-11-14 snapshot.

## Wave mechanics

golangci-lint v2.15's wrapcheck gained signature-based detection and flags errorfamily
constructors/clone methods as "unwrapped externals" in CONSUMER repos (the defining repo's own
package is exempt — wrapcheck never flags own-package returns). v2.15-pinned consumers solved it
already with three mechanisms: `ignore-package-globs: github.com/larsartmann/go-error-family*`
(cqrs-htmx), the `- go-error-family.` import-path sig with rationale comment (mr-sync), or
per-method sigs (httputil). Session 3 standardized the import-path sig for new ports.

## Actioned in session 3 (ported + verified green under local v2.15.0)

| Repo | Old pin | Clone sites | Outcome |
| --- | --- | --- | --- |
| go-codec | v2.14.0 | 2 | sig + pin → v2.15.0, CHANGELOG; ALL-GREEN |
| go-retry | v2.13.2 | 3 | sig + pin → v2.15.0, CHANGELOG; ALL-GREEN |
| go-composable-business-types | v2.14.0 | 1 | sig + pin → v2.15.0, CHANGELOG; ALL-GREEN (6 modules) |
| german-business-contract-automation | v2.13.2 | 33 | sig + pin → v2.15.0, CHANGELOG; ALL-GREEN |
| go-finding | v2.14.0 | 3 | ignore-sigs block added + pin → v2.15.0, CHANGELOG; ALL-GREEN (4 modules) |
| go-appkit | v2.14.0 | 1 | sig + pin → v2.15.0 ×2 sites, CHANGELOG; root + 13 submodules GREEN (integration fails on its own go.work pattern shape, pre-existing) |
| go-localsync | v2.13.2 | 2 | sig added to `ignore-sig-regexps`; NOT bumped — v2.15 also flags huma v2 sigs (per-repo triage) |
| KeyHolderAI | v2.14.0 | 93 | sig added; NOT bumped — pre-existing templ typecheck break (`templruntime.NewAttributeExpression` undefined) |
| Zlota44 | v2.13.2 | 103 | sig added; NOT bumped — same templ typecheck break class |
| legal-graph-ai-system | v2.14.0 | 2 | sig added; NOT bumped — pre-existing typecheck break (graphrag.NodeID type mismatch) |
| ChastityAPI | v2.13.2 | 5 | sig added; NOT bumped — mixed findings (http.Pusher.Push wrapcheck, noctx/paralleltest/exhaustruct test classes) |
| Standup-Killer | v2.13.2 | 2 | sig added; NOT bumped — 150+ findings under v2.15 (varnamelen/paralleltest/exhaustruct/wrapcheck/ireturn); own-session triage |

## Already solved before session 3 (v2.15-pinned, green)

cqrs-htmx (package globs), mr-sync (import-path sig — the convention source), cv* (see skip),
Code-To-CV-Agent, audiblesync, graph-features-language, google-domain-detector, go-filewatcher,
go-humanize-linter, project-meta, docs-gates, go-error-family + go-etag (session 2, this fleet row).

## Covered without action

- **upd** — `.*errorfamily\.` regexp sig already present.
- **go-auto-upgrade** — partial method sigs present; census shows only `.WithContext(` (covered).
- **golangci-lint-auto-configure** — package-glob regex covers go-error-family.
- **vision-review-agent** — no `.golangci.yml` at all; wrapcheck not enabled (no wave exposure).

## Skipped (parallel writers / not owned here)

- **cv / CV** — red at HEAD from an in-flight parallel-writer refactor (undefined
  `ExportToolbar`/`ExportFooterRow` in unpushed test code); both repos had unpushed commits.
  Revisit after that writer lands.
- **BuildFlow, buildflow, BuildFlow-pr44, ci-siblings, templates, template-\*, archived/\*** —
  owner's active tool repo, PR worktrees, scaffolds, mirrors.

## GOTOOLCHAIN audit (§f.22)

Pinned below go1.27.4 in workflows: art-dupl go1.27.1, go-linter-sdk go1.27.0, go-graph-rag
go1.27.1, go-output go1.27.2, crm go1.27.3, ratsgedaechtnis go1.27.2. NOT bumped (each repo's
own bump cadence; bumping pins fleet-wide without per-repo verification would trade one exposure
for ten breakages). Only go-etag and httputil were bumped (wave-1 scope, cue-vet-verified).
pim-sync already at go1.27.4. GO-2026-6859 exposure is governed by the local/CI toolchain, not
the lint pin — repos without GOTOOLCHAIN pins follow their runner's Go.
