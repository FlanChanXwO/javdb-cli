#!/bin/sh
# 检查文档路由与领域目录约定，避免 README 或贡献入口重新指向兼容 stub。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

for document in \
	"$repo_root/docs/index.md" \
	"$repo_root/docs/en/cli-reference.md" \
	"$repo_root/docs/en/sdk.md" \
	"$repo_root/docs/zh-CN/cli-reference.md" \
	"$repo_root/docs/zh-CN/sdk.md" \
	"$repo_root/docs/maintainers/architecture.md" \
	"$repo_root/docs/maintainers/development.md" \
	"$repo_root/docs/maintainers/agents/index.md" \
	"$repo_root/docs/maintainers/adr/0001-public-facade-and-domain-layout.md" \
	"$repo_root/AGENTS.md" \
	"$repo_root/CLAUDE.md" \
	"$repo_root/.github/copilot-instructions.md"; do
	test -s "$document"
done

grep -F 'docs/en/cli-reference.md' "$repo_root/README.md" >/dev/null
grep -F 'docs/en/sdk.md' "$repo_root/README.md" >/dev/null
grep -F 'docs/zh-CN/cli-reference.md' "$repo_root/README.zh-CN.md" >/dev/null
grep -F 'docs/zh-CN/sdk.md' "$repo_root/README.zh-CN.md" >/dev/null
grep -F 'internal/javdb/appapi' "$repo_root/AGENTS.md" >/dev/null
grep -F '@AGENTS.md' "$repo_root/CLAUDE.md" >/dev/null

if rg -n 'internal/(appapi|httpx|signature)' \
	"$repo_root/README.md" \
	"$repo_root/README.zh-CN.md" \
	"$repo_root/CONTRIBUTING.md" \
	"$repo_root/CONTRIBUTING.zh-CN.md" \
	"$repo_root/docs"; then
	printf '%s\n' 'stale internal package path found in documentation' >&2
	exit 1
fi
