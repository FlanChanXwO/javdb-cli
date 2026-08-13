#!/bin/sh
# 以固定目标和链接元数据构建一个可放入发布归档的 javdb 二进制。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
version=
target=
output=
release_date=

usage() {
	cat >&2 <<'EOF'
usage: scripts/build-release.sh --version VERSION --target OS/ARCH --output PATH [--release-date DATE]

Builds one of the six supported release targets with CGO disabled:
darwin/amd64, darwin/arm64, linux/amd64, linux/arm64,
windows/amd64, windows/arm64. --release-date (YYYY-MM-DD, from the audited
changelog) is embedded for the public root --version contract.
EOF
}

fail() {
	printf '%s\n' "release build: $*" >&2
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
		--output)
			[ "$#" -ge 2 ] || fail '--output requires a path'
			output=$2
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
[ -n "$output" ] || fail '--output is required'

case "$target" in
	darwin/amd64|darwin/arm64|linux/amd64|linux/arm64|windows/amd64|windows/arm64) ;;
	*) fail "unsupported target: $target" ;;
esac

goos=${target%/*}
goarch=${target#*/}
case "$goos" in
	windows) expected_binary=javdb.exe ;;
	*) expected_binary=javdb ;;
esac
[ "$(basename -- "$output")" = "$expected_binary" ] || fail "target $target requires output named $expected_binary"

output_parent=$(dirname -- "$output")
[ -d "$output_parent" ] || fail "output directory does not exist: $output_parent"

commit=${COMMIT:-$(git -C "$repo_root" rev-parse --short HEAD 2>/dev/null || printf '%s' unknown)}
build_date=${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}
module='github.com/FlanChanXwO/javdb-cli/internal/buildinfo'
ldflags="-s -w -X ${module}.Version=${version} -X ${module}.Commit=${commit} -X ${module}.BuildDate=${build_date}"
if [ -n "$release_date" ]; then
	case "$release_date" in
		20[0-9][0-9]-[0-9][0-9]-[0-9][0-9]) ;;
		*) fail "invalid --release-date: $release_date (expect YYYY-MM-DD)" ;;
	esac
	ldflags="$ldflags -X ${module}.ReleaseDate=${release_date}"
fi

printf 'building %s -> %s\n' "$target" "$output"
(
	cd "$repo_root"
	CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -buildvcs=false \
		-ldflags "$ldflags" \
		-o "$output" ./cmd/javdb
)
