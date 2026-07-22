#!/bin/sh
# 将已构建的单个平台 javdb 二进制封装为可发布归档。
set -eu

# macOS 的 cp 可能将扩展属性写为 AppleDouble 文件；发布包只应包含显式列出的文件。
export COPYFILE_DISABLE=1

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
binary=
target=
version=
output_dir=

usage() {
	cat >&2 <<'EOF'
usage: scripts/package-release.sh --binary PATH --target OS/ARCH --version VERSION --output-dir DIR

Packages exactly one platform binary plus LICENSE and README.md. Supported
targets: darwin/amd64, darwin/arm64, linux/amd64, linux/arm64,
windows/amd64, windows/arm64.
EOF
}

fail() {
	printf '%s\n' "package release: $*" >&2
	exit 1
}

# 在解析真实路径前逐级拒绝已有的符号链接，防止发布输出穿透到意外目录。
reject_output_symlink_ancestors() {
	case "$1" in
		/*) ancestor=$1 ;;
		*) ancestor=$PWD/$1 ;;
	esac
	while :; do
		[ ! -L "$ancestor" ] || fail "output directory contains a symlink ancestor: $ancestor"
		parent=$(dirname -- "$ancestor")
		[ "$parent" = "$ancestor" ] && break
		ancestor=$parent
	done
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		--binary)
			[ "$#" -ge 2 ] || fail '--binary requires a path'
			binary=$2
			shift 2
			;;
		--target)
			[ "$#" -ge 2 ] || fail '--target requires OS/ARCH'
			target=$2
			shift 2
			;;
		--version)
			[ "$#" -ge 2 ] || fail '--version requires a value'
			version=$2
			shift 2
			;;
		--output-dir)
			[ "$#" -ge 2 ] || fail '--output-dir requires a path'
			output_dir=$2
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

[ -n "$binary" ] || fail '--binary is required'
[ -n "$target" ] || fail '--target is required'
[ -n "$version" ] || fail '--version is required'
[ -n "$output_dir" ] || fail '--output-dir is required'
version=${version#v}

case "$target" in
	darwin/amd64|darwin/arm64|linux/amd64|linux/arm64)
		archive_ext=tar.gz
		expected_binary=javdb
		;;
	windows/amd64|windows/arm64)
		archive_ext=zip
		expected_binary=javdb.exe
		;;
	*)
		fail "unsupported target: $target"
		;;
esac

[ -f "$binary" ] || fail "binary is not a regular file: $binary"
[ ! -L "$binary" ] || fail "binary must not be a symlink: $binary"
[ "$(basename -- "$binary")" = "$expected_binary" ] || fail "target $target requires binary named $expected_binary"
[ -f "$repo_root/LICENSE" ] || fail 'root LICENSE is missing'
[ -f "$repo_root/README.md" ] || fail 'root README.md is missing'
[ -d "$output_dir" ] || fail "output directory does not exist: $output_dir"
[ ! -L "$output_dir" ] || fail "output directory must not be a symlink: $output_dir"
reject_output_symlink_ancestors "$output_dir"

output_dir=$(CDPATH= cd -- "$output_dir" && pwd)
target_os=${target%/*}
target_arch=${target#*/}
output="$output_dir/javdb-cli_${version}_${target_os}_${target_arch}.${archive_ext}"
[ ! -e "$output" ] || fail "output already exists: $output"

stage=$(mktemp -d "${TMPDIR:-/tmp}/javdb-release-package.XXXXXX")
temporary_output=$(mktemp "$output_dir/.javdb-cli-package.XXXXXX")
cleanup() {
	rm -rf "$stage"
	rm -f "$temporary_output"
}
trap cleanup EXIT HUP INT TERM

cp "$binary" "$stage/$expected_binary"
cp "$repo_root/LICENSE" "$stage/LICENSE"
cp "$repo_root/README.md" "$stage/README.md"
# macOS 的 zip 不覆盖 mktemp 预建空文件；移除后由归档器独占创建。
rm -f "$temporary_output"

case "$archive_ext" in
	tar.gz)
		tar -C "$stage" -czf "$temporary_output" "$expected_binary" LICENSE README.md
		;;
	zip)
		(
			cd "$stage"
			# Windows Git Bash 默认没有 zip，GitHub runner 预装的 7z 是唯一约定工具。
			case "$(uname -s)" in
				MINGW*|MSYS*|CYGWIN*)
					command -v 7z >/dev/null 2>&1 || fail 'Windows zip packaging requires runner-provided 7z'
					7z a -tzip -bd "$temporary_output" "$expected_binary" LICENSE README.md
					;;
				*)
					command -v zip >/dev/null 2>&1 || fail 'zip packaging requires zip'
					zip -q -r "$temporary_output" "$expected_binary" LICENSE README.md
					;;
			esac
		)
		;;
esac

mv -f "$temporary_output" "$output"
printf 'packaged %s\n' "$output"
