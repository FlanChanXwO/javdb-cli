#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$ROOT/build"
go build -o "$ROOT/build/javdb" "$ROOT/cmd/javdb"
echo "built $ROOT/build/javdb"
