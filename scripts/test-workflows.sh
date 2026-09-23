#!/bin/sh
# Verify stable CI/release trust boundaries without snapshot-testing YAML layout.
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
workflows="$repo_root/.github/workflows"
quality="$workflows/ci.yml"
platform="$workflows/platform-smoke.yml"
container="$workflows/container-smoke.yml"
release="$workflows/release.yml"
verification="$workflows/pr-verification.yml"
metadata="$workflows/pr-metadata.yml"
clawhub="$workflows/publish-clawhub.yml"

# Optional Quality work uses a job-level condition so GitHub renders the
# required Quality gate as skipped instead of a successful shell job.
grep -F 'needs: scope' "$quality" >/dev/null
grep -F "if: \${{ always() && (needs.scope.result != 'success' || needs.scope.outputs.quality_required == 'true') }}" "$quality" >/dev/null
grep -F 'name: Require successful scope classification' "$quality" >/dev/null
if grep -F 'steps.scope.outputs.docs_only' "$quality" >/dev/null; then
	echo 'Quality gate must use a real job-level skip' >&2
	exit 1
fi
grep -F "if: \${{ needs.dispatch.result == 'success' && needs.dispatch.outputs.execute == 'true' }}" "$verification" >/dev/null
for forbidden in 'display:"not required"' 'Mark verification as not required' 'matrix.required'; do
	if grep -F "$forbidden" "$verification" >/dev/null; then
		echo "verification workflow retains fake-skip compatibility path: $forbidden" >&2
		exit 1
	fi
done

# Untrusted pull_request code stays read-only. The trusted pull_request_target
# policy workflow owns worker dispatch and publishes the two stable smoke gates.
grep -F '  pull_request:' "$quality" >/dev/null
grep -F '  pull_request_target:' "$metadata" >/dev/null
grep -F 'actions: write' "$metadata" >/dev/null
grep -F 'checks: write' "$metadata" >/dev/null
grep -F 'platform-smoke.yml' "$metadata" >/dev/null
grep -F 'container-smoke.yml' "$metadata" >/dev/null
grep -F "publish_gate 'PR template gate'" "$metadata" >/dev/null
grep -F "publish_gate 'PR commands gate'" "$metadata" >/dev/null
if grep -F 'statuses: write' "$metadata" >/dev/null ||
	grep -F 'statuses/$HEAD_SHA' "$metadata" >/dev/null; then
	echo 'PR metadata gates must use app-bound Check Runs, not legacy commit statuses' >&2
	exit 1
fi
if grep -F 'return_run_details' "$metadata" >/dev/null; then
	echo '2026-03-10 workflow dispatch must not send removed return_run_details' >&2
	exit 1
fi
grep -F "steps.smoke_head.outcome == 'success'" "$metadata" >/dev/null
grep -F "steps.smoke_head.outcome == 'failure'" "$metadata" >/dev/null
grep -F 'git fetch --no-tags origin "+refs/pull/$PR/head:refs/remotes/pull/$PR/head"' "$metadata" >/dev/null
grep -F 'test "$(git rev-parse "refs/remotes/pull/$PR/head")" = "$HEAD_SHA"' "$metadata" >/dev/null
test "$(grep -Fc 'metadata_is_current || exit 0' "$metadata")" -eq 2
test "$(grep -Fc 'if ! cmp -s "$RUNNER_TEMP/event-pr-body-state.md" "$RUNNER_TEMP/current-pr-body-state.md"; then' "$metadata")" -eq 1

pending_line=$(grep -nF 'create_check "$context" in_progress' "$metadata" | head -1 | cut -d: -f1)
dispatch_line=$(grep -nF '"repos/$REPO/actions/workflows/$workflow/dispatches"' "$metadata" | head -1 | cut -d: -f1)
test -n "$pending_line"
test -n "$dispatch_line"
test "$pending_line" -lt "$dispatch_line"

if grep -F 'actions: write' "$quality" >/dev/null ||
	grep -F '/dispatches' "$quality" >/dev/null; then
	echo 'untrusted pull_request workflow must not dispatch smoke workers' >&2
	exit 1
fi

# Native/container matrices run through workflow_dispatch so their worker jobs
# do not become PR checks and execute untrusted PR code with read-only tokens.
grep -F '  workflow_dispatch:' "$platform" >/dev/null
grep -F '  workflow_dispatch:' "$container" >/dev/null
grep -F 'checks: write' "$platform" >/dev/null
grep -F 'checks: write' "$container" >/dev/null
test "$(grep -Fc 'ref: ${{ inputs.pr_number' "$platform")" -eq 1
test "$(grep -Fc 'ref: ${{ inputs.pr_number' "$container")" -eq 1
for worker in "$platform" "$container"; do
	grep -F 'name: Require the requested source' "$worker" >/dev/null
	grep -F 'test "$(git rev-parse HEAD)" = "$HEAD_SHA"' "$worker" >/dev/null
	grep -F '"repos/$REPO/check-runs/$CHECK_RUN_ID"' "$worker" >/dev/null
	grep -F -- '-f status=completed' "$worker" >/dev/null
	if grep -F 'statuses: write' "$worker" >/dev/null ||
		grep -F 'statuses/$HEAD_SHA' "$worker" >/dev/null; then
		echo "smoke worker must not retain commit-status compatibility publication: $worker" >&2
		exit 1
	fi
done
grep -F '"repos/$REPO/check-runs"' "$metadata" >/dev/null
grep -F "create_check 'Platform smoke gate' completed skipped" "$metadata" >/dev/null
grep -F "create_check 'Container smoke gate' completed skipped" "$metadata" >/dev/null
grep -F -- '--arg check_run_id "$check_run_id"' "$metadata" >/dev/null
if grep -F '  pull_request:' "$platform" >/dev/null ||
	grep -F '  pull_request:' "$container" >/dev/null; then
	echo 'smoke worker workflows must not emit pull-request checks' >&2
	exit 1
fi

# Shared platform registry remains the single source of platform sets.
grep -F './tools/platformmatrix --capability smoke' "$platform" >/dev/null
grep -F './tools/platformmatrix --capability container' "$container" >/dev/null
grep -F './tools/platformmatrix --capability verification' "$verification" >/dev/null
grep -F './tools/platformmatrix --capability release' "$release" >/dev/null

# Release artifacts are built once, bound to an immutable handoff, then reused.
test "$(grep -Fc 'sh scripts/build-platform.sh' "$release")" -eq 1
test "$(grep -Fc 'docker build' "$release")" -eq 1
release_dir_line=$(grep -nF 'mkdir -p release' "$release" | head -1 | cut -d: -f1)
handoff_line=$(grep -nF 'go run ./tools/release write-handoff' "$release" | head -1 | cut -d: -f1)
test -n "$release_dir_line" && test -n "$handoff_line" && test "$release_dir_line" -lt "$handoff_line"
grep -F 'write-handoff' "$release" >/dev/null
grep -F -- '--output release/release-handoff.json' "$release" >/dev/null
grep -F 'verify-handoff-set' "$release" >/dev/null

for publisher in "$workflows"/publish-*.yml; do
	grep -F 'workflow_run:' "$publisher" >/dev/null
	grep -F 'release_run_id:' "$publisher" >/dev/null
	grep -F 'release_run_id must be a positive decimal number' "$publisher" >/dev/null
	if grep -nE '(^|[^a-z-])(go build|docker build|build-platform\.sh|build-release\.sh|package-release)' "$publisher" >/dev/null; then
		echo "publisher must not rebuild release artifacts: $publisher" >&2
		exit 1
	fi
	if grep -nE 'latest successful run|latest release|inputs\.tag|latest_release' "$publisher" >/dev/null; then
		echo "publisher must not use fuzzy recovery inputs: $publisher" >&2
		exit 1
	fi
	if grep -nE '^[[:space:]]*(required_reviewers|reviewers):' "$publisher" >/dev/null; then
		echo "publisher must not add another approval boundary: $publisher" >&2
		exit 1
	fi
done

# The release pipeline has one human approval after preparation.
test "$(grep -Fc 'environment: release-approval' "$release")" -eq 1
approval_job=$(sed -n '/^  approve_release:$/,/^  publish:$/p' "$release")
printf '%s\n' "$approval_job" | grep -F 'needs: [prepare_release]' >/dev/null

# Publication lives in independent publishers, never in release.yml.
for forbidden in 'docker push' 'docker manifest create' 'docker manifest push' 'docker tag'; do
	if grep -F "$forbidden" "$release" >/dev/null; then
		echo "release.yml must not publish images inline: $forbidden" >&2
		exit 1
	fi
done
grep -F 'render-homebrew-formula.sh' "$workflows/publish-homebrew.yml" >/dev/null
if grep -F 'render-homebrew-formula.sh' "$release" >/dev/null; then
	echo 'Homebrew rendering must live in the dedicated publisher' >&2
	exit 1
fi

# ClawHub consumes the common handoff and exposes its token only at publish time.
grep -F 'verify-handoff-identity' "$clawhub" >/dev/null
clawhub_checkout_line=$(grep -nF 'uses: actions/checkout@' "$clawhub" | head -1 | cut -d: -f1)
clawhub_setup_go_line=$(grep -nF 'uses: actions/setup-go@' "$clawhub" | head -1 | cut -d: -f1)
clawhub_identity_line=$(grep -nF 'go run ./tools/release verify-handoff-identity' "$clawhub" | head -1 | cut -d: -f1)
test -n "$clawhub_checkout_line" && test -n "$clawhub_setup_go_line" && test -n "$clawhub_identity_line"
test "$clawhub_checkout_line" -lt "$clawhub_identity_line"
test "$clawhub_setup_go_line" -lt "$clawhub_identity_line"
grep -F 'clawhub@0.23.1' "$clawhub" >/dev/null
grep -F -- '--dry-run' "$clawhub" >/dev/null
test "$(grep -Fc 'CLAWHUB_TOKEN: ${{ secrets.CLAWHUB_TOKEN }}' "$clawhub")" -eq 1
if grep -F 'clawhub-release-tag' "$release" >/dev/null; then
	echo 'release.yml must not stage a ClawHub-specific handoff' >&2
	exit 1
fi

# Untrusted PR code cannot write reactions/comments; trusted policy owns dispatch.
verify_job=$(sed -n '/^  verify:$/,/^  aggregate:$/p' "$verification")
if printf '%s\n' "$verify_job" | grep -F 'issues: write' >/dev/null; then
	echo 'untrusted verification job must not have issue write permission' >&2
	exit 1
fi
printf '%s\n' "$verify_job" | grep -F 'contents: read' >/dev/null
grep -F '<!-- pr-test-result -->' "$verification" >/dev/null
grep -F -- '--check-trigger --comment-file' "$verification" >/dev/null
grep -F -- '--resolve' "$verification" >/dev/null
grep -F 'COMMANDS_JSON:' "$verification" >/dev/null
if grep -F "jq -r '.base.sha'" "$verification" >/dev/null; then
	echo 'PR verification must execute trusted code from the current base branch' >&2
	exit 1
fi
grep -F "jq -r '.base.ref'" "$verification" >/dev/null

# GitHub-owned actions remain pinned to immutable SHAs.
if grep -RhnE '^[[:space:]]*-[[:space:]]+uses:[[:space:]]+actions/[^@]+@' "$workflows" |
	grep -Ev '@[0-9a-f]{40}([[:space:]]|$)' >/dev/null; then
	echo 'GitHub-owned actions must be pinned to full commit SHAs' >&2
	exit 1
fi
