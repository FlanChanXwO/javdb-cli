#!/bin/sh
# 只验证 CI/Release 的稳定行为契约；具体 YAML 排版与实现细节交给 Actions 实际运行。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

for workflow in "$repo_root"/.github/workflows/*.yml; do
	ruby -e 'require "yaml"; YAML.load_file(ARGV.fetch(0))' "$workflow"
done

quality="$repo_root/.github/workflows/ci.yml"
platform="$repo_root/.github/workflows/platform-smoke.yml"
container="$repo_root/.github/workflows/container-smoke.yml"
release="$repo_root/.github/workflows/release.yml"
verification="$repo_root/.github/workflows/pr-verification.yml"
metadata="$repo_root/.github/workflows/pr-metadata.yml"

# Branch protection relies on these stable aggregate names.
grep -F 'name: Quality gate' "$quality" >/dev/null
grep -F 'name: Platform smoke gate' "$platform" >/dev/null
grep -F 'name: Container smoke gate' "$container" >/dev/null
grep -F "context='PR template gate'" "$metadata" >/dev/null
grep -F "context='PR test command gate'" "$metadata" >/dev/null

# Platform sets are resolved from the shared registry rather than copied into workflows.
grep -F './tools/platformmatrix --capability smoke' "$platform" >/dev/null
grep -F './tools/platformmatrix --capability container' "$container" >/dev/null
grep -F './tools/platformmatrix --capability verification' "$verification" >/dev/null
grep -F './tools/platformmatrix --capability release' "$release" >/dev/null
grep -F './tools/platformmatrix --capability homebrew' "$release" >/dev/null

# Release bytes are built once on fresh production runners, then prepared and approved.
grep -F 'build_production:' "$release" >/dev/null
grep -F 'Smoke the exact production package' "$release" >/dev/null
if grep -F 'Build, package, and smoke the versioned binary' "$release" >/dev/null; then
	echo 'release verification must not build a disposable second package' >&2
	exit 1
fi
grep -F 'prepare_release:' "$release" >/dev/null
grep -F 'go run ./tools/release verify-artifact-set' "$release" >/dev/null

# Exactly one human approval boundary. Publication environments must not require another approval.
test "$(grep -Fc 'environment: release-approval' "$release")" -eq 1
grep -F 'environment: release' "$release" >/dev/null
approval_line=$(grep -n '^  approve_release:$' "$release" | cut -d: -f1)
publish_line=$(grep -n '^  publish:$' "$release" | cut -d: -f1)
test "$approval_line" -lt "$publish_line"

# Approval must happen after the archive, container, and Homebrew verification work.
approval_job=$(sed -n '/^  approve_release:$/,/^  publish:$/p' "$release")
printf '%s\n' "$approval_job" | grep -F 'needs: [prepare_release, verify_homebrew_formula]' >/dev/null

# PR code execution cannot write PR comments/reactions; trusted jobs own those permissions.
verify_job=$(sed -n '/^  verify:$/,/^  aggregate:$/p' "$verification")
if printf '%s\n' "$verify_job" | grep -F 'issues: write' >/dev/null; then
	echo 'untrusted verification job must not have issue write permission' >&2
	exit 1
fi
printf '%s\n' "$verify_job" | grep -F 'contents: read' >/dev/null
grep -F '<!-- pr-test-result -->' "$verification" >/dev/null

# Trusted verifier code follows the current PR base branch tip rather than the PR's stale base commit.
if grep -F "jq -r '.base.sha'" "$verification" >/dev/null; then
	echo 'PR verification must not trust the PR-captured base.sha as the executor source' >&2
	exit 1
fi
grep -F "jq -r '.base.ref'" "$verification" >/dev/null
grep -F 'branches/$base_ref_encoded' "$verification" >/dev/null

sh "$repo_root/scripts/test-clawhub-publish-workflow.sh"
