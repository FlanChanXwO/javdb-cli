#!/bin/sh
# 核心平台构建契约：统一 primitive 产出的归档必须可直接解包并运行。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
temporary=$(mktemp -d "${TMPDIR:-/tmp}/javdb-platform-build-test.XXXXXX")
cleanup() {
	rm -rf "$temporary"
}
trap cleanup EXIT HUP INT TERM

version=0.0.0-platform-test
goos=$(go env GOOS)
goarch=$(go env GOARCH)
target="$goos/$goarch"

archive=$(
	cd "$repo_root"
	sh scripts/build-platform.sh \
		--version "$version" \
		--target "$target" \
		--output-dir "$temporary/dist"
)

expected="$temporary/dist/javdb-cli_${version}_${goos}_${goarch}"
binary="$temporary/smoke/javdb"
mkdir -p "$temporary/smoke"

case "$goos" in
	windows)
		expected="$expected.zip"
		binary="$temporary/smoke/javdb.exe"
		7z x "$expected" "-o$temporary/smoke" >/dev/null
		;;
	*)
		expected="$expected.tar.gz"
		tar -xzf "$expected" -C "$temporary/smoke"
		;;
esac

[ "$archive" = "$expected" ]
[ -f "$binary" ]
"$binary" --version | grep -Fx "javdb version $version"

# primitive 不维护平台白名单；registry 决定哪些 target 进入 CI。
portable_archive=$(
	cd "$repo_root"
	sh scripts/build-platform.sh \
		--version "$version" \
		--target linux/riscv64 \
		--output-dir "$temporary/portable"
)
[ "$portable_archive" = "$temporary/portable/javdb-cli_${version}_linux_riscv64.tar.gz" ]
tar -tzf "$portable_archive" | grep -Fx javdb >/dev/null

mkdir -p "$temporary/existing"
printf '%s\n' sentinel > "$temporary/existing/javdb"
if (
	cd "$repo_root"
	sh scripts/build-platform.sh \
		--version "$version" \
		--target linux/amd64 \
		--output-dir "$temporary/existing"
) >"$temporary/existing.out" 2>&1; then
	echo 'pre-existing binary unexpectedly overwritten' >&2
	exit 1
fi
[ "$(cat "$temporary/existing/javdb")" = sentinel ]
grep -F 'binary output already exists' "$temporary/existing.out" >/dev/null
