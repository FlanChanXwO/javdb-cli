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
grep -F "context='PR commands gate'" "$metadata" >/dev/null

# 用户可见 Check name 只回答「检查什么」：
#  - 每个 job 必须有显式 name，否则 GitHub 直接暴露内部 job id；
#  - setup job 不得使用内部实现词（Resolve ... matrix）。
# 注意：运行中的 matrix job 名称会被 GitHub 追加整个 matrix tuple，且被 skip 的 job
# 不会展开 name 中的表达式，所以只能约束 job id 与 setup 名称。
python3 - "$repo_root/.github/workflows" <<'PY'
import sys, pathlib, yaml
root = pathlib.Path(sys.argv[1])
failures = []
for path in sorted(root.glob('*.yml')):
    doc = yaml.safe_load(path.read_text())
    for job_id, spec in (doc.get('jobs') or {}).items():
        if not spec.get('name'):
            failures.append(f'{path.name}: job {job_id!r} has no user-facing name')
if failures:
    print('\n'.join(failures), file=sys.stderr)
    sys.exit(1)
PY

if grep -F 'Resolve platform matrix' "$platform" "$container" >/dev/null 2>&1; then
	echo 'setup jobs must use user-facing names' >&2
	exit 1
fi
grep -F 'name: Platform setup' "$platform" >/dev/null
grep -F 'name: Container setup' "$container" >/dev/null

# §12.1 aggregate 失败必须给出可操作原因，不能只留退出码。
grep -F 'Platform smoke failed on one or more required platforms.' "$platform" >/dev/null
grep -F 'Container smoke failed on one or more required platforms.' "$container" >/dev/null
grep -F 'Required quality checks did not complete successfully.' "$quality" >/dev/null
grep -F 'Platform configuration could not be resolved.' "$platform" >/dev/null

# §12.2 合法 skip 必须解释原因。
grep -F 'Documentation-only change; platform smoke is not required.' "$platform" >/dev/null
grep -F 'Container smoke is not required for this change.' "$container" >/dev/null

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
grep -F 'go run ./tools/release verify-source' "$release" >/dev/null

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

# Deleted trigger comments may disappear after /test is accepted. Only that
# concrete HTTP 404 becomes an empty reaction target; other API errors stay fatal.
# The trigger decision and the effective commands are owned by the single
# trusted policy binary; no second parser or shell-side scanning is allowed.
grep -F -- '--check-trigger --comment-file' "$verification" >/dev/null
grep -F -- '--resolve' "$verification" >/dev/null
if grep -F 'trimmed' "$verification" >/dev/null; then
	echo 'trigger detection must use the trusted policy, not shell trimming' >&2
	exit 1
fi

# A declared-but-invalid comment override must fail the /test run closed and
# must never fall back to the PR body commands.
grep -F 'OVERRIDE_DECLARED: ${{ steps.resolve.outputs.override_declared }}' "$verification" >/dev/null
grep -F 'if [ "$OVERRIDE_DECLARED" = true ]; then' "$verification" >/dev/null
grep -F 'fail_trigger "$RESOLVE_DESC"' "$verification" >/dev/null
if grep -F 'RESOLVE_DESC:-' "$verification" >/dev/null; then
	echo 'invalid overrides must not fall back to PR commands' >&2
	exit 1
fi

# The executed commands come from the resolver output, never from the PR-body
# gate, so a comment override actually changes what runs.
grep -F 'COMMANDS_JSON: ${{ needs.dispatch.outputs.commands }}' "$verification" >/dev/null
grep -F 'commands: ${{ steps.resolve.outputs.effective_commands }}' "$verification" >/dev/null
grep -F 'verification_hash: ${{ steps.resolve.outputs.effective_hash }}' "$verification" >/dev/null

# Verification identity binds PR number, HEAD SHA, and the effective commands
# hash so an override is a distinct execution identity (§6).
grep -F 'verification_identity()' "$verification" >/dev/null
grep -F "printf '%s:%s:%s'" "$verification" >/dev/null
grep -F '"$PR" "$HEAD_SHA" "$VERIFICATION_HASH"' "$verification" >/dev/null

test "$(grep -Fc '(HTTP 404)' "$verification")" -eq 2
test "$(grep -Fc '*"(HTTP 404)"*) return 0 ;;' "$verification")" -eq 2
test "$(grep -Fc '[ -n "$subject_id" ] || return 0' "$verification")" -eq 4
if grep -F 'subject_id=$(reaction_subject "$1") || return 0' "$verification" >/dev/null; then
	echo 'reaction lookup must not broadly suppress non-404 failures' >&2
	exit 1
fi
if grep -F -- '-f content="$2" >/dev/null 2>&1 || true' "$verification" >/dev/null; then
	echo 'GraphQL reaction failures must remain observable' >&2
	exit 1
fi

# Cancelling a superseded run is best-effort, but failures must remain visible.
if grep -F 'gh api --method POST "repos/$REPO/actions/runs/$old_run/cancel" >/dev/null 2>&1 || true' "$verification" >/dev/null; then
	echo 'superseded run cancellation must not silently discard failures' >&2
	exit 1
fi
grep -F 'if cancel_output=$(gh api --method POST "repos/$REPO/actions/runs/$old_run/cancel" 2>&1); then' "$verification" >/dev/null
grep -F 'failed to cancel superseded verification run %s: %s' "$verification" >/dev/null
grep -F '"$old_run" "$cancel_output" >&2' "$verification" >/dev/null

# §25.3：旧 Real API e2e orchestration 已被统一 Verification 取代，不得长期
# 同时存在两套入口。这里禁止旧入口文件与引用回归。
for legacy in "$repo_root/.github/workflows/e2e.yml" "$repo_root/e2e/run.sh"; do
	if [ -e "$legacy" ]; then
		echo "legacy e2e entrypoint still present: $legacy" >&2
		exit 1
	fi
done
if [ -d "$repo_root/e2e" ] && [ -n "$(find "$repo_root/e2e" -type f -print -quit)" ]; then
	echo 'legacy e2e directory still contains files' >&2
	exit 1
fi
# 排除本脚本自身：它必须写出旧路径才能完成检查。
if grep -rn --exclude-dir=.git --exclude-dir='goal-*' --exclude-dir=.worktrees \
	--exclude=test-workflows.sh \
	-e 'e2e/run\.sh' -e 'workflows/e2e\.yml' \
	"$repo_root/.github" "$repo_root/scripts" "$repo_root/docs" 2>/dev/null; then
	echo 'legacy e2e entrypoint is still referenced' >&2
	exit 1
fi

sh "$repo_root/scripts/test-clawhub-publish-workflow.sh"
