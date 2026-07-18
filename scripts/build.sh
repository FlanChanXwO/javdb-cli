#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$ROOT/build"

VERSION="${VERSION:-dev}"
# Prefer git metadata when available
COMMIT="${COMMIT:-$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
MODULE="github.com/FlanChanXwO/javdb-cli/internal/buildinfo"

LDFLAGS=(
  "-s" "-w"
  "-X" "${MODULE}.Version=${VERSION}"
  "-X" "${MODULE}.Commit=${COMMIT}"
  "-X" "${MODULE}.BuildDate=${BUILD_DATE}"
)

CGO_ENABLED=0 go build -trimpath -ldflags "${LDFLAGS[*]}" \
  -o "$ROOT/build/javdb" "$ROOT/cmd/javdb"
echo "built $ROOT/build/javdb (version=${VERSION} commit=${COMMIT})"
