#!/bin/sh
# 固定公开 facade 与 CLI 的依赖方向，避免协议 adapter 再次泄漏到命令层。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

test -d "$repo_root/internal/javdb/appapi"
test -d "$repo_root/internal/javdb/protocol/httpx"
test -d "$repo_root/internal/javdb/protocol/signature"
test ! -e "$repo_root/internal/appapi"
test ! -e "$repo_root/internal/httpx"
test ! -e "$repo_root/internal/signature"

if rg -n 'internal/javdb/(appapi|protocol)' \
	"$repo_root/cmd" \
	"$repo_root/internal/cli" \
	-g '*.go'; then
	printf '%s\n' 'CLI or binary entry imports a JavDB protocol implementation directly' >&2
	exit 1
fi
