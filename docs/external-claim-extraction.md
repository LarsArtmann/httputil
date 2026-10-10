# External-Claim Raw Extraction Patterns

Proven fall-backs for verifying claims about EXTERNAL projects when
AI-summarizing tools fail (the 2026-09-15 sessions; recorded so the next
`agentic_fetch` outage costs nothing). Rule of record (AGENTS.md): never
encode an external claim into docs/code/reviews without raw-source
verification.

## 1. Raw markdown of a repo file: `gh api` contents endpoint

AI wrappers mangle or refuse; the raw file never lies.

```bash
gh api repos/<owner>/<repo>/contents/<path>/README.md --jq '.content' | base64 -d
```

Variants: `--jq '.encoding'` to confirm base64; append `?ref=<tag>` to pin
a release (`gh api "repos/o/r/contents/README.md?ref=v1.2.0"`).

## 2. Large specs and version claims: `download` + grep

Pages that choke summarizers (RFCs, long changelogs, spec drafts) go to
disk first, then targeted greps answer the actual claim:

1. `download` the URL to a temp file.
2. `grep -n -i '<the exact claim terms>' /tmp/spec.txt`.
3. Cite the line numbers, not the summary.

For claims of the form "feature X landed in version Y", grep the project's
own CHANGELOG for the version heading and the feature terms in the same
range — a claim that survives both greps is citable with them.

## 3. Browser-version claims: `mdn/browser-compat-data` JSON

"Chrome supports X since version N" class claims verify against
machine-readable compat data, not blog posts:

```bash
gh api repos/mdn/browser-compat-data/contents/<feature-path>.json --jq '.content' | base64 -d | jq '.["<feature>"].__compat.support.chrome'
```

The `version_added` field is the citable fact. Cross-check a second claim
source only when the JSON is silent (removed/preview flags).

## Known tooling failure

`agentic_fetch` can fail with a json-unmarshal error on some pages
(observed 2026-09-15; environment tooling, not this repo). When it does,
do NOT retry it — switch directly to pattern 1 or 2 above. A fix belongs
in the tool's own repository, not here.
