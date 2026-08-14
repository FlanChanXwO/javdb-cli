#!/bin/sh
# 静态检查 CI 与发布 workflow 的关键结构，避免未运行到 Actions 才发现 YAML 或目标矩阵漂移。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

for workflow in \
	"$repo_root/.github/workflows/ci.yml" \
	"$repo_root/.github/workflows/platform-smoke.yml" \
	"$repo_root/.github/workflows/release.yml" \
	"$repo_root/.github/workflows/e2e.yml" \
	"$repo_root/.github/workflows/publish-clawhub.yml"; do
	ruby -e 'require "yaml"; YAML.load_file(ARGV.fetch(0))' "$workflow"
done

change_scope_action="$repo_root/.github/actions/classify-change-scope/action.yml"
ruby -e 'require "yaml"; YAML.load_file(ARGV.fetch(0))' "$change_scope_action"

platform_workflow="$repo_root/.github/workflows/platform-smoke.yml"
quality_workflow="$repo_root/.github/workflows/ci.yml"
release_workflow="$repo_root/.github/workflows/release.yml"

for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
	grep -F "goos: ${target%/*}" "$platform_workflow" >/dev/null
	grep -F "goarch: ${target#*/}" "$platform_workflow" >/dev/null
	grep -F "goos: ${target%/*}" "$release_workflow" >/dev/null
	grep -F "goarch: ${target#*/}" "$release_workflow" >/dev/null
done

grep -F 'ref: ${{ env.RELEASE_TAG }}' "$release_workflow" >/dev/null
grep -F 'release_notes_audit:' "$release_workflow" >/dev/null
grep -F 'pull-requests: read' "$release_workflow" >/dev/null
grep -F 'needs: [validate, verify_release_source, release_notes_audit]' "$release_workflow" >/dev/null
grep -F 'scripts/releasenotes render' "$release_workflow" >/dev/null
grep -F -- '--notes-file release/release-notes.md' "$release_workflow" >/dev/null
grep -F 'gh release create "$RELEASE_TAG"' "$release_workflow" >/dev/null
grep -F 'HOMEBREW_TAP_DEPLOY_ENABLED' "$release_workflow" >/dev/null
# publish job 必须绑定受保护的 release environment，只在此处读取签名私钥，
# 并从已验证 archives 生成 manifest、signature 与由 manifest 派生的 checksums。
grep -F 'environment: release' "$release_workflow" >/dev/null
grep -F 'JAVDB_RELEASE_ED25519_PRIVATE_KEYS: ${{ secrets.JAVDB_RELEASE_ED25519_PRIVATE_KEYS }}' "$release_workflow" >/dev/null
grep -F 'go run ./scripts/sign-release' "$release_workflow" >/dev/null
grep -F 'dist/release-manifest.json' "$release_workflow" >/dev/null
grep -F 'dist/release-manifest.sig' "$release_workflow" >/dev/null
grep -F 'dist/checksums.txt' "$release_workflow" >/dev/null
sh "$repo_root/scripts/test-clawhub-publish-workflow.sh"

# 文档专属路径必须有一致的 classifier 与稳定汇总 gate；workflow 本身的改动不在白名单内。
for workflow in "$quality_workflow" "$platform_workflow"; do
	grep -F 'classify_changes:' "$workflow" >/dev/null
	grep -F './.github/actions/classify-change-scope' "$workflow" >/dev/null
done
grep -F 'go run ./scripts/changescope' "$change_scope_action" >/dev/null
grep -F 'Validate pull request release-note declaration' "$quality_workflow" >/dev/null
grep -F 'docs_only' "$quality_workflow" >/dev/null
grep -F 'platform_smoke_gate:' "$platform_workflow" >/dev/null
grep -F 'name: Platform smoke gate' "$platform_workflow" >/dev/null
grep -F 'MATRIX_RESULT' "$platform_workflow" >/dev/null
# 六个受保护的 matrix check 必须在文档 PR 上同样展开；只允许把昂贵步骤切成轻量确认。
grep -F 'name: Packaged binary smoke ${{ matrix.goos }}/${{ matrix.goarch }}' "$platform_workflow" >/dev/null
grep -F "runs-on: \${{ needs.classify_changes.outputs.docs_only == 'true' && 'ubuntu-24.04' || matrix.runner }}" "$platform_workflow" >/dev/null
grep -F 'Confirm docs-only native smoke skip' "$platform_workflow" >/dev/null
if grep -Eq '^    if:.*docs_only' "$platform_workflow"; then
	echo 'platform smoke matrix must not use a docs-only job-level condition' >&2
	exit 1
fi

# e2e workflow 必须排除纯文档/changelog/skills 路径,避免文档改动触发真实 API 冒烟。
e2e_workflow="$repo_root/.github/workflows/e2e.yml"
grep -F "paths-ignore:" "$e2e_workflow" >/dev/null
grep -F "'docs/**'" "$e2e_workflow" >/dev/null
grep -F "'CHANGELOG.md'" "$e2e_workflow" >/dev/null
grep -F "'skills/**'" "$e2e_workflow" >/dev/null
grep -F 'secrets.JAVDB_E2E_USERNAME' "$e2e_workflow" >/dev/null
