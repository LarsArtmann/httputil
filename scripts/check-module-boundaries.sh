#!/usr/bin/env bash
# Verifies each Go module in the workspace stands alone (the consumer view):
# with GOWORK=off, every module must build and vet against its own go.mod —
# including the replace directives that make local paths resolvable.
set -euo pipefail

cd "$(dirname "$0")/.."

# /mnt/buildcache-style environments may pin an unwritable GOCACHE; fall back.
if [ -z "${GOCACHE:-}" ] || ! mkdir -p "$GOCACHE" 2>/dev/null; then
	export GOCACHE="$HOME/.cache/go-build-httputil"
fi

if [ -z "${GOMODCACHE:-}" ] || ! mkdir -p "$GOMODCACHE" 2>/dev/null; then
	export GOMODCACHE="$HOME/go/pkg/mod"
fi

modules=("." "./server_timing")

status=0
for mod in "${modules[@]}"; do
	echo "=== module: $mod"
	(
		cd "$mod"
		export GOWORK=off
		go build ./... || status=1
		go vet ./... || status=1
	)
done

# go.work must list every local module with a go.mod.
for mod in "${modules[@]}"; do
	if [ "$mod" != "." ] && ! grep -q "$mod" go.work; then
		echo "go.work is missing module: $mod"
		status=1
	fi
done

if [ "$status" -eq 0 ]; then
	echo "module boundaries OK"
else
	echo "module boundary check FAILED" >&2
	exit 1
fi
