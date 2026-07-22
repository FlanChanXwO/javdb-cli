#!/bin/sh
# 构建当前本机开发二进制；发布目标使用 build-release.sh 的显式目标契约。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
mkdir -p "$repo_root/build"

version=${VERSION:-dev}
# Git 元数据不可用时保留真实的 unknown，而非把构建伪装成某个提交。
commit=${COMMIT:-$(git -C "$repo_root" rev-parse --short HEAD 2>/dev/null || printf '%s' unknown)}
build_date=${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}
module='github.com/FlanChanXwO/javdb-cli/internal/buildinfo'
ldflags="-s -w -X ${module}.Version=${version} -X ${module}.Commit=${commit} -X ${module}.BuildDate=${build_date}"
output="$repo_root/build/javdb"
if [ "$(go env GOOS)" = windows ]; then
	output="$output.exe"
fi

CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags "$ldflags" \
	-o "$output" "$repo_root/cmd/javdb"
printf 'built %s (version=%s commit=%s)\n' "$output" "$version" "$commit"
