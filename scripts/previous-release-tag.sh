#!/bin/sh
# 输出指定版本对应的上一稳定发布 tag；空输出表示首个发布版本。
set -eu

if [ "$#" -ne 1 ]; then
	printf '%s\n' 'usage: previous-release-tag.sh vX.Y.Z' >&2
	exit 2
fi

release_tag=$1
if ! printf '%s' "$release_tag" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$'; then
	printf 'release tag must be stable SemVer vX.Y.Z, got: %s\n' "$release_tag" >&2
	exit 2
fi

# 正式发布时只考虑目标 tag 的祖先；本地验证未打 tag 的历史 changelog 时，
# 使用本地 tag 列表作为离线 fallback，避免删除过的临时 tag 阻断历史文档校验。
if git rev-parse --verify --quiet "$release_tag^{commit}" >/dev/null 2>&1; then
	tags=$(git tag --merged "$release_tag" --list 'v[0-9]*' --sort=-v:refname)
else
	tags=$(git tag --list 'v[0-9]*' --sort=-v:refname)
fi

while IFS= read -r tag; do
	[ -n "$tag" ] || continue
	[ "$tag" = "$release_tag" ] && continue
	if [ -s "changelog/$tag/en.md" ] && [ -s "changelog/$tag/zh-CN.md" ]; then
		printf '%s\n' "$tag"
		exit 0
	fi
done <<EOF
$tags
EOF
