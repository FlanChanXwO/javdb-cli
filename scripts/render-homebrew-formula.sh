#!/usr/bin/env bash
# Render Homebrew formula from release checksums.txt
# Usage:
#   VERSION=0.1.0 bash scripts/render-homebrew-formula.sh \
#     --checksums dist/checksums.txt \
#     --output /tmp/javdb-cli.rb
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="${VERSION:-}"
CHECKSUMS=""
OUTPUT=""
REPO="${REPO:-FlanChanXwO/javdb-cli}"
TEMPLATE="${TEMPLATE:-$ROOT/templates/homebrew/javdb-cli.rb.tmpl}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --checksums) CHECKSUMS="$2"; shift 2 ;;
    --output) OUTPUT="$2"; shift 2 ;;
    --template) TEMPLATE="$2"; shift 2 ;;
    --repo) REPO="$2"; shift 2 ;;
    --version) VERSION="$2"; shift 2 ;;
    *) echo "unknown arg: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$VERSION" || -z "$CHECKSUMS" || -z "$OUTPUT" ]]; then
  echo "need VERSION, --checksums, --output" >&2
  exit 1
fi
VERSION="${VERSION#v}"

sha_for() {
  local os="$1" arch="$2" ext="$3"
  local name="javdb-cli_${VERSION}_${os}_${arch}.${ext}"
  # checksums.txt lines: "<sha>  <file>" or "<sha> *<file>"
  local line
  line="$(grep -E "[ /]${name}\$" "$CHECKSUMS" || true)"
  if [[ -z "$line" ]]; then
    echo "missing checksum for $name in $CHECKSUMS" >&2
    exit 1
  fi
  echo "$line" | awk '{print $1}'
}

url_for() {
  local os="$1" arch="$2" ext="$3"
  echo "https://github.com/${REPO}/releases/download/v${VERSION}/javdb-cli_${VERSION}_${os}_${arch}.${ext}"
}

DARWIN_AMD64_SHA256="$(sha_for darwin amd64 tar.gz)"
DARWIN_ARM64_SHA256="$(sha_for darwin arm64 tar.gz)"
LINUX_AMD64_SHA256="$(sha_for linux amd64 tar.gz)"
LINUX_ARM64_SHA256="$(sha_for linux arm64 tar.gz)"

DARWIN_AMD64_URL="$(url_for darwin amd64 tar.gz)"
DARWIN_ARM64_URL="$(url_for darwin arm64 tar.gz)"
LINUX_AMD64_URL="$(url_for linux amd64 tar.gz)"
LINUX_ARM64_URL="$(url_for linux arm64 tar.gz)"

mkdir -p "$(dirname "$OUTPUT")"
sed \
  -e "s|{{VERSION}}|${VERSION}|g" \
  -e "s|{{FORMULA_CLASS}}|JavdbCli|g" \
  -e "s|{{DARWIN_AMD64_URL}}|${DARWIN_AMD64_URL}|g" \
  -e "s|{{DARWIN_AMD64_SHA256}}|${DARWIN_AMD64_SHA256}|g" \
  -e "s|{{DARWIN_ARM64_URL}}|${DARWIN_ARM64_URL}|g" \
  -e "s|{{DARWIN_ARM64_SHA256}}|${DARWIN_ARM64_SHA256}|g" \
  -e "s|{{LINUX_AMD64_URL}}|${LINUX_AMD64_URL}|g" \
  -e "s|{{LINUX_AMD64_SHA256}}|${LINUX_AMD64_SHA256}|g" \
  -e "s|{{LINUX_ARM64_URL}}|${LINUX_ARM64_URL}|g" \
  -e "s|{{LINUX_ARM64_SHA256}}|${LINUX_ARM64_SHA256}|g" \
  "$TEMPLATE" > "$OUTPUT"

echo "wrote $OUTPUT"
