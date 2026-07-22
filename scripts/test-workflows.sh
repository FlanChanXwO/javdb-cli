#!/bin/sh
# 静态检查 CI 与发布 workflow 的关键结构，避免未运行到 Actions 才发现 YAML 或目标矩阵漂移。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

for workflow in \
	"$repo_root/.github/workflows/ci.yml" \
	"$repo_root/.github/workflows/platform-smoke.yml" \
	"$repo_root/.github/workflows/release.yml"; do
	ruby -e 'require "yaml"; YAML.load_file(ARGV.fetch(0))' "$workflow"
done

platform_workflow="$repo_root/.github/workflows/platform-smoke.yml"
release_workflow="$repo_root/.github/workflows/release.yml"

for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
	grep -F "goos: ${target%/*}" "$platform_workflow" >/dev/null
	grep -F "goarch: ${target#*/}" "$platform_workflow" >/dev/null
	grep -F "goos: ${target%/*}" "$release_workflow" >/dev/null
	grep -F "goarch: ${target#*/}" "$release_workflow" >/dev/null
done

grep -F 'ref: ${{ env.RELEASE_TAG }}' "$release_workflow" >/dev/null
grep -F 'gh release create "$RELEASE_TAG"' "$release_workflow" >/dev/null
grep -F 'HOMEBREW_TAP_DEPLOY_ENABLED' "$release_workflow" >/dev/null
