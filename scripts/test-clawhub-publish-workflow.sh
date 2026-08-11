#!/bin/sh
# 检查 ClawHub 发布 workflow 的信任边界与 token 最小暴露范围。
set -eu

script_dir=$(dirname -- "$0")
repo_root=$(CDPATH=; cd -- "$script_dir/.." && pwd)
workflow_dir="$repo_root/.github/workflows"
clawhub_workflow="$workflow_dir/publish-clawhub.yml"
release_workflow="$workflow_dir/release.yml"
skill="$repo_root/skills/javdb-cli/SKILL.md"
github_expr='$'

for workflow in "$clawhub_workflow" "$release_workflow"; do
	ruby -e 'require "yaml"; YAML.load_file(ARGV.fetch(0))' "$workflow"
done

ruby - "$clawhub_workflow" "$release_workflow" <<'RUBY'
pattern = %r{\Aactions/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)?@[0-9a-f]{40}\z}
ARGV.each do |path|
  File.foreach(path).with_index(1) do |line, number|
    next unless line =~ /^\s*-\s+uses:\s+(\S+)/
    reference = Regexp.last_match(1)
    next unless reference.start_with?("actions/")
    abort("#{path}:#{number}: action is not pinned to a full commit SHA: #{reference}") unless pattern.match?(reference)
  end
end
RUBY

grep -F 'workflow_run:' "$clawhub_workflow" >/dev/null
grep -F -- '- Release' "$clawhub_workflow" >/dev/null
grep -F 'types:' "$clawhub_workflow" >/dev/null
grep -F 'permissions: {}' "$clawhub_workflow" >/dev/null
grep -F 'actions: read' "$clawhub_workflow" >/dev/null
grep -F 'contents: read' "$clawhub_workflow" >/dev/null
grep -F 'name: clawhub-release-tag' "$clawhub_workflow" >/dev/null
grep -F "path: $github_expr{{ runner.temp }}/clawhub-release-tag" "$clawhub_workflow" >/dev/null
grep -F 'handoff_dir="$RUNNER_TEMP/clawhub-release-tag"' "$clawhub_workflow" >/dev/null
grep -F 'git merge-base --is-ancestor' "$clawhub_workflow" >/dev/null
grep -F "releases/tags/\$RELEASE_TAG" "$clawhub_workflow" >/dev/null
grep -F "ref: $github_expr{{ steps.release_tag.outputs.value }}" "$clawhub_workflow" >/dev/null

grep -F 'clawhub@0.23.1' "$clawhub_workflow" >/dev/null
grep -F 'clawhub skill publish skills/javdb-cli' "$clawhub_workflow" >/dev/null
grep -F -- "--source-commit \"\$RELEASE_COMMIT\"" "$clawhub_workflow" >/dev/null
grep -F "CLAWHUB_TOKEN: $github_expr{{ secrets.CLAWHUB_TOKEN }}" "$clawhub_workflow" >/dev/null
grep -F -- '--dry-run' "$clawhub_workflow" >/dev/null

test "$(grep -c "CLAWHUB_TOKEN: $github_expr{{ secrets.CLAWHUB_TOKEN }}" "$clawhub_workflow")" = 1
grep -F 'name: Hand off the immutable release tag to ClawHub' "$release_workflow" >/dev/null
grep -F 'clawhub-release-tag/release-tag' "$release_workflow" >/dev/null

for field in slug version displayName summary license homepage tags name description; do
	grep -Eq "^$field: " "$skill" || {
		printf '%s\n' "skill metadata field missing: $field" >&2
		exit 1
	}
done
