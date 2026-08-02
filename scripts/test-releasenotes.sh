#!/bin/sh
# 离线验证版本化发布说明、受版本控制的 release-prep manifest 与根目录兼容入口。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

test -s "$repo_root/changelog/README.md"
test -s "$repo_root/changelog/README.zh-CN.md"
test -s "$repo_root/changelog/unreleased/en.md"
test -s "$repo_root/changelog/unreleased/zh-CN.md"
grep -F 'changelog/README.md' "$repo_root/CHANGELOG.md" >/dev/null
grep -F 'changelog/README.zh-CN.md' "$repo_root/CHANGELOG.zh-CN.md" >/dev/null

for plan in "$repo_root"/changelog/plans/v*.json; do
	test -s "$plan"
	version=$(basename "$plan" .json)
	version=${version#v}
	previous=$(python3 - "$plan" "$version" <<'PY'
import json, sys
plan = json.load(open(sys.argv[1]))
if plan.get("version") != sys.argv[2]:
    raise SystemExit(f"manifest version mismatch: {plan.get('version')!r}")
previous = plan.get("previous_tag")
compare = plan.get("compare_url")
if previous is None:
    if sys.argv[2] != "0.1.0" or compare is not None:
        raise SystemExit("only v0.1.0 may omit previous_tag and compare_url")
    print("")
else:
    expected = f"https://github.com/FlanChanXwO/javdb-cli/compare/{previous}...v{sys.argv[2]}"
    if compare != expected:
        raise SystemExit(f"manifest compare_url mismatch: {compare!r}, want {expected!r}")
    print(previous)
PY
	)
	set -- validate --version "$version" --dir "$repo_root/changelog/v$version"
	if [ -n "$previous" ]; then
		set -- "$@" --previous "$previous"
	fi
	go run "$repo_root/scripts/releasenotes" "$@"
done
