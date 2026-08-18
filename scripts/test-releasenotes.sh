#!/bin/sh
# 离线验证版本化双语发布说明和根目录兼容入口；发布说明直接来自 changelog/vX.Y.Z。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

test -s "$repo_root/changelog/README.md"
test -s "$repo_root/changelog/README.zh-CN.md"
test -s "$repo_root/changelog/unreleased/en.md"
test -s "$repo_root/changelog/unreleased/zh-CN.md"
test ! -e "$repo_root/changelog/plans"
grep -F 'changelog/README.md' "$repo_root/CHANGELOG.md" >/dev/null
grep -F 'changelog/README.zh-CN.md' "$repo_root/CHANGELOG.zh-CN.md" >/dev/null

found_version=false
for english in "$repo_root"/changelog/v*/en.md; do
	test -s "$english"
	directory=$(dirname "$english")
	version=$(basename "$directory")
	version=${version#v}
	chinese="$directory/zh-CN.md"
	test -s "$chinese"
	grep -F "| [v$version](" "$repo_root/changelog/README.md" >/dev/null
	grep -F "| [v$version](" "$repo_root/changelog/README.zh-CN.md" >/dev/null

	previous=$(CDPATH= cd -- "$repo_root" && sh scripts/previous-release-tag.sh "v$version")
	set -- validate --version "$version" --dir "$directory"
	if [ -n "$previous" ]; then
		set -- "$@" --previous "$previous"
	fi
	(CDPATH= cd "$repo_root" && go run ./scripts/releasenotes "$@")
	found_version=true
done

test "$found_version" = true
