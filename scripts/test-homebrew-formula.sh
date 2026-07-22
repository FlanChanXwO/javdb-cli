#!/bin/sh
# 用固定 checksum fixture 验证 Formula 渲染，不访问 GitHub Release 或 Homebrew tap。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
temporary=$(mktemp -d "$repo_root/.javdb-homebrew-formula-test.XXXXXX")
trap 'rm -rf "$temporary"' EXIT HUP INT TERM

version=0.1.1
checksums="$temporary/checksums.txt"
formula="$temporary/javdb-cli.rb"

printf '%s\n' \
	"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  javdb-cli_${version}_darwin_amd64.tar.gz" \
	"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb  javdb-cli_${version}_darwin_arm64.tar.gz" \
	"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc  javdb-cli_${version}_linux_amd64.tar.gz" \
	"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd  javdb-cli_${version}_linux_arm64.tar.gz" \
	"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee  javdb-cli_${version}_windows_amd64.zip" \
	"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff  javdb-cli_${version}_windows_arm64.zip" \
	> "$checksums"

bash "$repo_root/scripts/render-homebrew-formula.sh" \
	--version "$version" \
	--checksums "$checksums" \
	--output "$formula"
ruby -c "$formula" >/dev/null

grep -F 'class JavdbCli < Formula' "$formula" >/dev/null
grep -F 'version "0.1.1"' "$formula" >/dev/null
grep -F 'https://github.com/FlanChanXwO/javdb-cli/releases/download/v0.1.1/javdb-cli_0.1.1_darwin_arm64.tar.gz' "$formula" >/dev/null
grep -F 'sha256 "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"' "$formula" >/dev/null
grep -F 'bin.install "javdb"' "$formula" >/dev/null
grep -F 'assert_equal "v#{version}", version_info["version"]' "$formula" >/dev/null

if rg -i 'windows|depends_on' "$formula" >/dev/null; then
	printf '%s\n' 'Formula unexpectedly selects Windows or declares a build dependency' >&2
	exit 1
fi
