#!/bin/sh
# 检查 ClawHub/SkillHub 发布 workflow 的信任边界与 token 最小暴露范围。
set -eu

script_dir=$(dirname -- "$0")
repo_root=$(CDPATH=; cd -- "$script_dir/.." && pwd)
workflow_dir="$repo_root/.github/workflows"
clawhub_workflow="$workflow_dir/publish-clawhub.yml"
skillhub_workflow="$workflow_dir/publish-skillhub.yml"
release_workflow="$workflow_dir/release.yml"
skill="$repo_root/skills/javdb-cli/SKILL.md"
github_expr='$'

for workflow in "$clawhub_workflow" "$skillhub_workflow" "$release_workflow"; do
	ruby -e 'require "yaml"; YAML.load_file(ARGV.fetch(0))' "$workflow"
done

ruby - "$clawhub_workflow" "$skillhub_workflow" "$release_workflow" <<'RUBY'
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

for workflow in "$clawhub_workflow" "$skillhub_workflow"; do
	grep -F 'workflow_run:' "$workflow" >/dev/null
	grep -F -- '- Release' "$workflow" >/dev/null
	grep -F 'types:' "$workflow" >/dev/null
	grep -F 'permissions: {}' "$workflow" >/dev/null
	grep -F 'actions: read' "$workflow" >/dev/null
	grep -F 'contents: read' "$workflow" >/dev/null
	grep -F 'name: skillhub-release-tag' "$workflow" >/dev/null
	grep -F 'git merge-base --is-ancestor' "$workflow" >/dev/null
	grep -F "releases/tags/\$RELEASE_TAG" "$workflow" >/dev/null
	grep -F "ref: $github_expr{{ steps.release_tag.outputs.value }}" "$workflow" >/dev/null
done

grep -F 'clawhub@0.23.1' "$clawhub_workflow" >/dev/null
grep -F 'clawhub skill publish skills/javdb-cli' "$clawhub_workflow" >/dev/null
grep -F -- "--source-commit \"\$RELEASE_COMMIT\"" "$clawhub_workflow" >/dev/null
grep -F "CLAWHUB_TOKEN: $github_expr{{ secrets.CLAWHUB_TOKEN }}" "$clawhub_workflow" >/dev/null
grep -F -- '--dry-run' "$clawhub_workflow" >/dev/null

grep -F 'SKILLHUB_HOST: https://api.skillhub.cn' "$skillhub_workflow" >/dev/null
grep -F -- '--skip-self-upgrade publish skills/javdb-cli' "$skillhub_workflow" >/dev/null
grep -F "SKILLHUB_TOKEN: $github_expr{{ secrets.SKILLHUB_TOKEN }}" "$skillhub_workflow" >/dev/null
grep -F -- '--dry-run' "$skillhub_workflow" >/dev/null

test "$(grep -c "CLAWHUB_TOKEN: $github_expr{{ secrets.CLAWHUB_TOKEN }}" "$clawhub_workflow")" = 1
test "$(grep -c "env -u CLAWHUB_TOKEN CLAWHUB_CONFIG_PATH=\"\$public_config_path\"" "$clawhub_workflow")" = 2
test "$(grep -c "SKILLHUB_TOKEN: $github_expr{{ secrets.SKILLHUB_TOKEN }}" "$skillhub_workflow")" = 1
grep -F 'name: Hand off the immutable release tag to skill publishers' "$release_workflow" >/dev/null
grep -F 'skillhub-release-tag/release-tag' "$release_workflow" >/dev/null

for field in slug version displayName summary license homepage tags name description; do
	grep -Eq "^$field: " "$skill" || {
		printf '%s\n' "skill metadata field missing: $field" >&2
		exit 1
	}
done
