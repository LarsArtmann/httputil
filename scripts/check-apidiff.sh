#!/usr/bin/env bash
# check-apidiff.sh BASELINE_ZIP MODULE_DIR MODULE_LABEL BASELINE_REF
#
# Machine check of the frozen-API promise (docs/v1-stability.md): fails when
# apidiff reports incompatible exported-API changes between the committed
# baseline (the latest released tag's export data, see apidiff/README.md) and
# the current tree. apidiff exits 0 even when it reports incompatible changes
# (verified 2026-11-02), so a non-empty report — not the exit code — is the
# failure condition. Used by scripts/prerelease-check.sh (gate 3b) and the
# ci.yml apidiff job; the tool pin lives here and in apidiff/README.md only.
set -uo pipefail

baseline_zip=${1:?usage: check-apidiff.sh BASELINE_ZIP MODULE_DIR MODULE_LABEL BASELINE_REF}
module_dir=${2:?usage: check-apidiff.sh BASELINE_ZIP MODULE_DIR MODULE_LABEL BASELINE_REF}
label=${3:?usage: check-apidiff.sh BASELINE_ZIP MODULE_DIR MODULE_LABEL BASELINE_REF}
baseline_ref=${4:?usage: check-apidiff.sh BASELINE_ZIP MODULE_DIR MODULE_LABEL BASELINE_REF}

repo_root="$(cd "$(dirname "$0")/.." && pwd)" || exit 1
case "$baseline_zip" in
/*) ;;
*) baseline_zip="$repo_root/$baseline_zip" ;;
esac

report="$(cd "$repo_root/$module_dir" && GOWORK=off go run golang.org/x/exp/cmd/apidiff@v0.0.0-20261014143641-f0251eb4aa3e -m -incompatible "$baseline_zip" .)" || {
	printf 'apidiff failed to run for %s (baseline %s): is %s present and readable, and is the toolchain healthy?\n' "$label" "$baseline_ref" "$baseline_zip" >&2
	exit 1
}

if [[ -n "$report" ]]; then
	printf 'INCOMPATIBLE exported-API changes in %s vs baseline %s:\n%s\n' "$label" "$baseline_ref" "$report" >&2
	exit 1
fi

printf '%s: no incompatible changes vs %s\n' "$label" "$baseline_ref"
