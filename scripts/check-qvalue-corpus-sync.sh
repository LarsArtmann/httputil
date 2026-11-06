#!/usr/bin/env bash
# Enforces that the shared q-value corpus exists in both places and stays
# byte-identical. The root module's testdata/qvalue-corpus.txt is canonical; the
# endpoints sub-module carries a mirror because a nested module directory is
# excluded from the root module zip, so the sub-module must ship the corpus
# itself. Both modules' tests assert the SAME expectations, which is what keeps
# the two deliberately duplicated parsers from drifting apart silently.
set -euo pipefail

cd "$(dirname "$0")/.."

canonical="testdata/qvalue-corpus.txt"
mirror="endpoints/testdata/qvalue-corpus.txt"

for file in "$canonical" "$mirror"; do
	if [ ! -f "$file" ]; then
		echo "q-value corpus missing: $file" >&2
		exit 1
	fi
done

if ! cmp -s "$canonical" "$mirror"; then
	echo "q-value corpus drift: $mirror must be a byte-identical copy of $canonical" >&2
	echo "re-copy it with: cp $canonical $mirror" >&2
	diff -u "$canonical" "$mirror" >&2 || true
	exit 1
fi

echo "q-value corpus sync OK ($canonical == $mirror)"
