#!/bin/sh
# 统一单个平台的发布构建与规范归档；测试、smoke、release 决定何时调用。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
version=
target=
output_dir=
release_date=

usage() {
	cat >&2 <<'EOF'
usage: scripts/build-platform.sh --version VERSION --target OS/ARCH --output-dir DIR [--release-date DATE]
EOF
}

fail() {
	printf '%s\n' "platform build: $*" >&2
	exit 1
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--version)
			[ "$#" -ge 2 ] || fail '--version requires a value'
			version=$2
			shift 2
			;;
		--target)
			[ "$#" -ge 2 ] || fail '--target requires OS/ARCH'
			target=$2
			shift 2
			;;
		--output-dir)
			[ "$#" -ge 2 ] || fail '--output-dir requires a path'
			output_dir=$2
			shift 2
			;;
		--release-date)
			[ "$#" -ge 2 ] || fail '--release-date requires a value'
			release_date=$2
			shift 2
			;;
		-h|--help)
			usage
			exit 0
			;;
		*)
			usage
			fail "unknown argument: $1"
			;;
	esac
done

[ -n "$version" ] || fail '--version is required'
[ -n "$target" ] || fail '--target is required'
[ -n "$output_dir" ] || fail '--output-dir is required'
version=${version#v}

case "$target" in
	*/*)
		goos=${target%%/*}
		goarch=${target#*/}
		;;
	*) fail 'target must use OS/ARCH form' ;;
esac
[ -n "$goos" ] && [ -n "$goarch" ] || fail 'target must contain non-empty OS and ARCH'
case "$goarch" in */*) fail 'target must contain exactly one slash' ;; esac
case "$goos" in
	windows)
		binary_name=javdb.exe
		archive_ext=zip
		;;
	*)
		binary_name=javdb
		archive_ext=tar.gz
		;;
esac

mkdir -p "$output_dir"
output_dir=$(CDPATH= cd -- "$output_dir" && pwd)
binary="$output_dir/$binary_name"
# primitive 的 EXIT cleanup 会删除本次临时 binary；拒绝既有路径，避免误删调用者原有文件。
if [ -e "$binary" ] || [ -L "$binary" ]; then
	fail "binary output already exists: $binary"
fi
cleanup() {
	rm -f "$binary"
}
trap cleanup EXIT HUP INT TERM

if [ -n "$release_date" ]; then
	sh "$repo_root/scripts/build-release.sh" \
		--version "$version" \
		--release-date "$release_date" \
		--target "$target" \
		--output "$binary" >&2
else
	sh "$repo_root/scripts/build-release.sh" \
		--version "$version" \
		--target "$target" \
		--output "$binary" >&2
fi

sh "$repo_root/scripts/package-release.sh" \
	--binary "$binary" \
	--version "$version" \
	--target "$target" \
	--output-dir "$output_dir" >&2

printf '%s\n' "$output_dir/javdb-cli_${version}_${goos}_${goarch}.${archive_ext}"
