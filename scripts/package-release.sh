#!/usr/bin/env bash
# Build multi-arch release archives for javdb.
# Usage:
#   VERSION=0.1.0 bash scripts/package-release.sh
# Output under dist/:
#   javdb-cli_${VERSION}_${os}_${arch}.tar.gz  (zip on windows)
#   checksums.txt
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-}"
if [[ -z "$VERSION" ]]; then
  if git describe --tags --exact-match HEAD >/dev/null 2>&1; then
    VERSION="$(git describe --tags --exact-match HEAD)"
    VERSION="${VERSION#v}"
  else
    echo "VERSION is required (e.g. VERSION=0.1.0)" >&2
    exit 1
  fi
fi
VERSION="${VERSION#v}"

COMMIT="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
MODULE="github.com/FlanChanXwO/javdb-cli/internal/buildinfo"
OUT="${OUT_DIR:-$ROOT/dist}"
mkdir -p "$OUT"
rm -f "$OUT"/javdb-cli_"${VERSION}"_* "$OUT"/checksums.txt 2>/dev/null || true

LDFLAGS=(
  "-s" "-w"
  "-X" "${MODULE}.Version=${VERSION}"
  "-X" "${MODULE}.Commit=${COMMIT}"
  "-X" "${MODULE}.BuildDate=${BUILD_DATE}"
)

TARGETS=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
  "windows arm64"
)

for target in "${TARGETS[@]}"; do
  # shellcheck disable=SC2086
  set -- $target
  GOOS="$1"
  GOARCH="$2"
  EXT=""
  [[ "$GOOS" == "windows" ]] && EXT=".exe"
  BIN="javdb${EXT}"
  STAGE="$OUT/stage_${GOOS}_${GOARCH}"
  rm -rf "$STAGE"
  mkdir -p "$STAGE"

  echo "building ${GOOS}/${GOARCH}..."
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build -trimpath \
    -ldflags "${LDFLAGS[*]}" \
    -o "$STAGE/$BIN" ./cmd/javdb

  cp LICENSE "$STAGE/" 2>/dev/null || true
  cp README.md "$STAGE/" 2>/dev/null || true

  ARCHIVE="javdb-cli_${VERSION}_${GOOS}_${GOARCH}"
  if [[ "$GOOS" == "windows" ]]; then
    (cd "$STAGE" && zip -qr "$OUT/${ARCHIVE}.zip" .)
  else
    tar -C "$STAGE" -czf "$OUT/${ARCHIVE}.tar.gz" .
  fi
  rm -rf "$STAGE"
done

(
  cd "$OUT"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum javdb-cli_"${VERSION}"_* > checksums.txt
  else
    shasum -a 256 javdb-cli_"${VERSION}"_* > checksums.txt
  fi
)

echo "artifacts in $OUT:"
ls -la "$OUT"
