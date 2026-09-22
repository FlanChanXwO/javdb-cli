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

# Branch protection relies on these aggregate check names.
grep -F 'name: Quality gate' "$quality" >/dev/null
grep -F 'name: Platform smoke gate' "$quality" >/dev/null
grep -F 'name: Container smoke gate' "$quality" >/dev/null
grep -F "context='PR template gate'" "$metadata" >/dev/null
grep -F "context='PR commands gate'" "$metadata" >/dev/null

# PRs expose only the two aggregate smoke gates; native/container matrices run
# through workflow_dispatch so their worker jobs do not become PR checks.
grep -F '  pull_request:' "$quality" >/dev/null
grep -F '  workflow_dispatch:' "$platform" >/dev/null
grep -F '  workflow_dispatch:' "$container" >/dev/null
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

# Retired orchestration and one-off probes stay retired.
for obsolete in \
	"$workflows/e2e.yml" \
	"$repo_root/e2e/run.sh" \
	"$workflows/auto-assign.yml" \
	"$repo_root/.github/auto_assign.yml" \
	"$repo_root/scripts/login_probe.go"; do
	if [ -e "$obsolete" ]; then
		echo "obsolete repository entrypoint still present: $obsolete" >&2
		exit 1
	fi
done

# Quality invokes maintained repository checks directly. pre-commit is local-only.
if grep -F 'pre_commit run --all-files' "$quality" >/dev/null; then
	echo 'Quality must not rerun repository checks through pre-commit' >&2
	exit 1
fi
grep -F 'sh scripts/test-releasenotes.sh' "$quality" >/dev/null
