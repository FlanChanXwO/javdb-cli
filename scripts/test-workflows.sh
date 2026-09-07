#!/bin/sh
# 静态检查 CI 与发布 workflow 的关键结构，避免未运行到 Actions 才发现 YAML 或目标矩阵漂移。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

for workflow in \
	"$repo_root/.github/workflows/ci.yml" \
	"$repo_root/.github/workflows/platform-smoke.yml" \
	"$repo_root/.github/workflows/release.yml" \
	"$repo_root/.github/workflows/e2e.yml" \
	"$repo_root/.github/workflows/publish-clawhub.yml" \
	"$repo_root/.github/workflows/container-smoke.yml" \
	"$repo_root/.github/workflows/auto-assign.yml" \
	"$repo_root/.github/workflows/pr-triage.yml"; do
	ruby -e 'require "yaml"; YAML.load_file(ARGV.fetch(0))' "$workflow"
done

ruby -e 'require "yaml"; ARGV.each { |path| YAML.load_file(path) }' \
	"$repo_root/.github/auto_assign.yml" \
	"$repo_root/.github/labeler.yml"

change_scope_action="$repo_root/.github/actions/classify-change-scope/action.yml"
ruby -e 'require "yaml"; YAML.load_file(ARGV.fetch(0))' "$change_scope_action"

platform_workflow="$repo_root/.github/workflows/platform-smoke.yml"
quality_workflow="$repo_root/.github/workflows/ci.yml"
release_workflow="$repo_root/.github/workflows/release.yml"

# quality 的 release-note hook 需要历史 tags；不能只依赖默认浅 checkout。
quality_checkout=$(sed -n '/^  quality:/,/^      - uses: actions\/setup-go@/p' "$quality_workflow")
printf '%s\n' "$quality_checkout" | grep -F 'fetch-depth: 0' >/dev/null

for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
	grep -F "goos: ${target%/*}" "$platform_workflow" >/dev/null
	grep -F "goarch: ${target#*/}" "$platform_workflow" >/dev/null
	grep -F "goos: ${target%/*}" "$release_workflow" >/dev/null
	grep -F "goarch: ${target#*/}" "$release_workflow" >/dev/null
done

grep -F 'ref: ${{ env.RELEASE_TAG }}' "$release_workflow" >/dev/null
grep -F 'release_notes_audit:' "$release_workflow" >/dev/null
# Dockerfile 契约：pinned digest 基础镜像、非 root javdb 用户、状态目录与 /work、固定入口。
dockerfile="$repo_root/Dockerfile"
grep -F 'FROM debian@sha256:' "$dockerfile" >/dev/null
grep -F 'useradd --home-dir /home/javdb --create-home --shell /usr/sbin/nologin --uid 1000 javdb' "$dockerfile" >/dev/null
grep -F 'mkdir -p /home/javdb/.javdb-cli' "$dockerfile" >/dev/null
grep -F 'COPY dist/javdb /usr/local/bin/javdb' "$dockerfile" >/dev/null
grep -F 'COPY LICENSE /usr/share/doc/javdb-cli/' "$dockerfile" >/dev/null
grep -F 'ENV HOME=/home/javdb' "$dockerfile" >/dev/null
grep -F 'WORKDIR /work' "$dockerfile" >/dev/null
grep -F 'LABEL org.opencontainers.image.source="https://github.com/FlanChanXwO/javdb-cli"' "$dockerfile" >/dev/null
grep -F 'USER javdb' "$dockerfile" >/dev/null
sed -n 's/^ENTRYPOINT //p' "$dockerfile" | grep -F '["/usr/local/bin/javdb"]' >/dev/null
# 容器 smoke workflow 在 Dockerfile/.dockerignore 变更时验证镜像运行时契约。
container_workflow="$repo_root/.github/workflows/container-smoke.yml"
grep -F 'name: Container image smoke' "$container_workflow" >/dev/null
grep -F "'Dockerfile'" "$container_workflow" >/dev/null
grep -F "'.dockerignore'" "$container_workflow" >/dev/null
grep -F 'javdb version 0.0.0-smoke (2026-01-01)' "$container_workflow" >/dev/null
grep -F '/home/javdb/.javdb-cli/' "$container_workflow" >/dev/null
grep -F 'test -s /usr/share/doc/javdb-cli/LICENSE' "$container_workflow" >/dev/null
grep -F 'pull-requests: read' "$release_workflow" >/dev/null
grep -F 'scripts/previous-release-tag.sh' "$release_workflow" >/dev/null
grep -F 'sh scripts/test-releasenotes.sh' "$release_workflow" >/dev/null
grep -F 'needs: [validate, verify_release_source, release_notes_audit]' "$release_workflow" >/dev/null
grep -F 'scripts/releasenotes render' "$release_workflow" >/dev/null
grep -F -- '--notes-file release/release-notes.md' "$release_workflow" >/dev/null
grep -F 'gh release create "$RELEASE_TAG"' "$release_workflow" >/dev/null
grep -F 'HOMEBREW_TAP_DEPLOY_ENABLED' "$release_workflow" >/dev/null
# 容器镜像发布：build_container 从同一不可变 tag 重建 Linux 二进制、构建并验证镜像契约，
# 上传 docker save tar 供恢复重跑；publish_container 只消费已验证镜像推 GHCR 并发布多架构 manifest。
grep -F 'build_container:' "$release_workflow" >/dev/null
grep -F 'publish_container:' "$release_workflow" >/dev/null
grep -F 'needs: [build_container, publish]' "$release_workflow" >/dev/null
grep -F 'packages: write' "$release_workflow" >/dev/null
grep -F 'verified-container-${{ matrix.artifact }}' "$release_workflow" >/dev/null
grep -F 'ghcr.io/flanchanxwo/javdb-cli' "$release_workflow" >/dev/null
grep -F 'docker manifest create "ghcr.io/flanchanxwo/javdb-cli:${RELEASE_TAG}"' "$release_workflow" >/dev/null
grep -F 'docker manifest create "ghcr.io/flanchanxwo/javdb-cli:latest"' "$release_workflow" >/dev/null
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
if grep -F 'Validate pull request release-note declaration' "$quality_workflow" >/dev/null; then
	echo 'quality workflow must not validate PR release-note metadata' >&2
	exit 1
fi
grep -F 'docs_only' "$quality_workflow" >/dev/null
grep -F 'platform_smoke_gate:' "$platform_workflow" >/dev/null
grep -F 'name: Platform smoke gate' "$platform_workflow" >/dev/null
grep -F "if: needs.classify_changes.outputs.docs_only != 'true'" "$platform_workflow" >/dev/null
grep -F 'MATRIX_RESULT' "$platform_workflow" >/dev/null
# 代码变更保留六个平台原生矩阵；文档变更由聚合 gate 接管，job 名称不得暴露未展开的 matrix 表达式。
grep -F 'name: Packaged binary smoke' "$platform_workflow" >/dev/null
if grep -F 'name: Packaged binary smoke ${{ matrix.goos }}/${{ matrix.goarch }}' "$platform_workflow" >/dev/null; then
	echo 'platform smoke job name must not expose an unexpanded matrix expression' >&2
	exit 1
fi
grep -F 'runs-on: ${{ matrix.runner }}' "$platform_workflow" >/dev/null
if grep -F 'Confirm docs-only native smoke skip' "$platform_workflow" >/dev/null; then
	echo 'platform smoke must not create docs-only placeholder checks' >&2
	exit 1
fi

# e2e workflow 必须排除纯文档/changelog/skills 路径,避免文档改动触发真实 API 冒烟。
e2e_workflow="$repo_root/.github/workflows/e2e.yml"
grep -F "paths-ignore:" "$e2e_workflow" >/dev/null
grep -F "'docs/**'" "$e2e_workflow" >/dev/null
grep -F "'CHANGELOG.md'" "$e2e_workflow" >/dev/null
grep -F "'skills/**'" "$e2e_workflow" >/dev/null
grep -F 'secrets.JAVDB_E2E_USERNAME' "$e2e_workflow" >/dev/null

# PR 自动化只使用目标仓库权限，并固定第三方 action 的不可变提交。
auto_assign_workflow="$repo_root/.github/workflows/auto-assign.yml"
triage_workflow="$repo_root/.github/workflows/pr-triage.yml"
grep -F 'pull_request_target:' "$auto_assign_workflow" >/dev/null
grep -F 'pull-requests: write' "$auto_assign_workflow" >/dev/null
grep -F 'kentaro-m/auto-assign-action@f4648c0a9fdb753479e9e75fc251f507ce17bb7e' "$auto_assign_workflow" >/dev/null
grep -F 'pull_request_target:' "$triage_workflow" >/dev/null
grep -F 'issues: write' "$triage_workflow" >/dev/null
grep -F 'actions/labeler@8558fd74291d67161a8a78ce36a881fa63b766a9' "$triage_workflow" >/dev/null
grep -F 'configuration-path: .github/labeler.yml' "$triage_workflow" >/dev/null
grep -F 'sync-labels: true' "$triage_workflow" >/dev/null
