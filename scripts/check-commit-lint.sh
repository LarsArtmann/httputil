#!/usr/bin/env bash
# check-commit-lint.sh — reject commits whose subject lacks a conventional
# commit prefix (feat, fix, docs, style, refactor, perf, test, build, ci,
# chore, revert, release), optionally with a scope and/or ! marker.
#
# Usage: check-commit-lint.sh <base-ref> <head-ref>
# Example: ./scripts/check-commit-lint.sh origin/master origin/HEAD
set -euo pipefail

base="${1:?usage: check-commit-lint.sh <base-ref> <head-ref>}"
head="${2:?usage: check-commit-lint.sh <base-ref> <head-ref>}"

pattern='^(feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert|release)(\([^)]+\))?!?: '

status=0

while IFS= read -r sha; do
	[[ -z "$sha" ]] && continue

	subject=$(git log -1 --format=%s "$sha")

	if ! grep -Eq "$pattern" <<<"$subject"; then
		echo "commit $sha lacks a conventional prefix: $subject" >&2
		status=1
	fi
done < <(git rev-list --no-merges "${base}..${head}")

exit "$status"
