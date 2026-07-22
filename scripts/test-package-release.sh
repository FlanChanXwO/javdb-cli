#!/bin/sh
# 验证发布归档的成员与关键失败路径，不依赖网络或本机 Go 工具链。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
temporary=$(mktemp -d "$repo_root/.javdb-package-release-test.XXXXXX")
trap 'rm -rf "$temporary"' EXIT HUP INT TERM
mkdir -p "$temporary/out"

create_binary() {
	path=$1
	printf '#!/bin/sh\nprintf fixture\n' > "$path"
	chmod 0755 "$path"
}

assert_members() {
	archive=$1
	format=$2
	actual="$temporary/actual-$format.txt"
	case "$format" in
		tar.gz)
			tar -tzf "$archive" | while IFS= read -r entry; do
				case "$entry" in */) ;; *) printf '%s\n' "$entry" ;; esac
			done | LC_ALL=C sort > "$actual"
			;;
		zip)
			unzip -Z1 "$archive" | while IFS= read -r entry; do
				case "$entry" in */) ;; *) printf '%s\n' "$entry" ;; esac
			done | LC_ALL=C sort > "$actual"
			;;
	esac
	printf '%s\n' LICENSE README.md "$expected_binary" | LC_ALL=C sort > "$temporary/expected.txt"
	diff -u "$temporary/expected.txt" "$actual"
}

darwin_binary="$temporary/javdb"
create_binary "$darwin_binary"
expected_binary=javdb
sh "$repo_root/scripts/package-release.sh" \
	--binary "$darwin_binary" --target darwin/arm64 --version 0.1.1 --output-dir "$temporary/out"
assert_members "$temporary/out/javdb-cli_0.1.1_darwin_arm64.tar.gz" tar.gz

windows_binary="$temporary/javdb.exe"
create_binary "$windows_binary"
expected_binary=javdb.exe
sh "$repo_root/scripts/package-release.sh" \
	--binary "$windows_binary" --target windows/amd64 --version v0.1.1 --output-dir "$temporary/out"
assert_members "$temporary/out/javdb-cli_0.1.1_windows_amd64.zip" zip

expect_failure() {
	expected=$1
	shift
	if output=$("$@" 2>&1); then
		printf '%s\n' "command unexpectedly succeeded: $*" >&2
		exit 1
	fi
	printf '%s\n' "$output" | grep -F "$expected" >/dev/null || {
		printf '%s\n' "failure did not explain $expected: $output" >&2
		exit 1
	}
}

expect_failure 'unsupported target: plan9/amd64' \
	sh "$repo_root/scripts/package-release.sh" --binary "$darwin_binary" --target plan9/amd64 --version 0.1.1 --output-dir "$temporary/out"
expect_failure 'requires binary named javdb.exe' \
	sh "$repo_root/scripts/package-release.sh" --binary "$darwin_binary" --target windows/arm64 --version 0.1.1 --output-dir "$temporary/out"
expect_failure 'output already exists' \
	sh "$repo_root/scripts/package-release.sh" --binary "$darwin_binary" --target darwin/arm64 --version 0.1.1 --output-dir "$temporary/out"
