#!/bin/sh
# 固定公开 facade 与 CLI 的依赖方向，避免协议 adapter 再次泄漏到命令层。
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

for required_dir in \
	sdk \
	internal/common/jsonx \
	internal/common/scalar \
	internal/cli/invocation \
	internal/cli/client \
	internal/cli/authstore \
	internal/cli/result \
	internal/cli/entity \
	internal/cli/commands/auth \
	internal/cli/commands/config \
	internal/cli/commands/search \
	internal/cli/commands/detail \
	internal/cli/commands/comments \
	internal/cli/commands/magnets \
	internal/cli/commands/download \
	internal/cli/commands/tags \
	internal/cli/commands/browse \
	internal/cli/commands/actor \
	internal/cli/commands/series \
	internal/cli/commands/maker \
	internal/cli/commands/director \
	internal/cli/commands/code \
	internal/cli/commands/list \
	internal/cli/commands/watched \
	internal/cli/commands/want \
	internal/cli/commands/recent \
	internal/cli/commands/collections \
	internal/cli/commands/mark \
	internal/cli/commands/unmark \
	internal/cli/commands/rankings \
	internal/cli/commands/top250 \
	internal/cli/commands/lists \
	internal/cli/commands/update \
	internal/cli/commands/version \
	internal/javdb/appapi \
	internal/javdb/appapi/client \
	internal/javdb/appapi/model \
	internal/javdb/appapi/codec \
	internal/javdb/appapi/media \
	internal/javdb/appapi/endpoint/auth \
	internal/javdb/appapi/endpoint/browse \
	internal/javdb/appapi/endpoint/entity \
	internal/javdb/appapi/endpoint/lists \
	internal/javdb/appapi/endpoint/magnets \
	internal/javdb/appapi/endpoint/movie \
	internal/javdb/appapi/endpoint/rankings \
	internal/javdb/appapi/endpoint/route \
	internal/javdb/appapi/endpoint/search \
	internal/javdb/appapi/endpoint/user \
	internal/javdb/protocol/httpx \
	internal/javdb/protocol/signature \
	internal/update/model \
	internal/update/archive \
	internal/update/release \
	internal/update/source \
	internal/update/process \
	internal/config/paths \
	internal/config/settings \
	internal/storage/auth \
	internal/storage/tags \
	internal/storage/route \
	scripts/releasenotes \
	scripts/internal/releasenotes/model \
	scripts/internal/releasenotes/github \
	scripts/internal/releasenotes/audit \
	scripts/internal/releasenotes/document \
	scripts/internal/releasenotes/prepare \
	scripts/internal/releasenotes/history; do
	if [ ! -d "$repo_root/$required_dir" ]; then
		printf 'missing required domain directory: %s\n' "$required_dir" >&2
		exit 1
	fi
done

test ! -e "$repo_root/javdb"
test ! -e "$repo_root/internal/appapi"
test ! -e "$repo_root/internal/httpx"
test ! -e "$repo_root/internal/signature"
test ! -e "$repo_root/scripts/internal/releasenotesrender"

# release-note compat seam 已删除：main 只做分派，领域实现位于 scripts/internal/releasenotes/*。
test ! -e "$repo_root/scripts/releasenotes/compat.go"
if [ ! -f "$repo_root/scripts/releasenotes/main.go" ]; then
	printf '%s\n' 'missing release-note entry point: scripts/releasenotes/main.go' >&2
	exit 1
fi

# CLI 根包只保留 root.go 与 root_test.go；facade/input/output/app/movie/magnet 与旧分组命令已删除。
if [ ! -f "$repo_root/internal/cli/root.go" ]; then
	printf '%s\n' 'missing real CLI root: internal/cli/root.go' >&2
	exit 1
fi
if [ ! -f "$repo_root/internal/cli/root_test.go" ]; then
	printf '%s\n' 'missing CLI root contract test: internal/cli/root_test.go' >&2
	exit 1
fi
# 根目录除 root.go/root_test.go 外不得有其他 Go 文件。
if find "$repo_root/internal/cli" -maxdepth 1 -name '*.go' | rg -v '/(root\.go|root_test\.go)$' | rg -q .; then
	printf '%s\n' 'internal/cli root contains Go files other than root.go/root_test.go' >&2
	exit 1
fi
test ! -e "$repo_root/internal/cli/facade.go"
test ! -e "$repo_root/internal/cli/root"
test ! -e "$repo_root/internal/cli/input"
test ! -e "$repo_root/internal/cli/output"
test ! -e "$repo_root/internal/cli/app"
test ! -e "$repo_root/internal/cli/movie"
test ! -e "$repo_root/internal/cli/magnet"
test ! -e "$repo_root/internal/cli/commands/account"
test ! -e "$repo_root/internal/cli/commands/catalog"
test ! -e "$repo_root/internal/cli/commands/user"

# 每个命令目录必须有与目录同名的主文件；禁止泛化主文件。
for cmd_dir in \
	auth config search detail comments magnets download tags browse \
	actor series maker director code list \
	watched want recent collections mark unmark \
	rankings top250 lists update version; do
	if [ ! -f "$repo_root/internal/cli/commands/$cmd_dir/$cmd_dir.go" ]; then
		printf 'missing command main file: %s.go\n' "$cmd_dir" >&2
		exit 1
	fi
done
if find "$repo_root/internal/cli/commands" -maxdepth 2 \( -name 'command.go' -o -name 'output.go' -o -name 'printer.go' -o -name 'render.go' \) | rg -q .; then
	printf '%s\n' 'command directories contain a generic main file' >&2
	exit 1
fi

# App API 根包是真实 Client 组合层（client.go），不再保留 facade.go。
if [ ! -f "$repo_root/internal/javdb/appapi/client.go" ]; then
	printf '%s\n' 'missing real App API root Client composition: internal/javdb/appapi/client.go' >&2
	exit 1
fi
test ! -e "$repo_root/internal/javdb/appapi/facade.go"

# internal/config 根目录不建立 Go package。
if find "$repo_root/internal/config" -maxdepth 1 -name '*.go' | rg -q .; then
	printf '%s\n' 'internal/config root must not contain Go files' >&2
	exit 1
fi

# internal/update 根包只保留 Coordinator、其最小接口与测试；facade 与根 alias types 已删除。
for update_file in \
	internal/update/coordinator.go \
	internal/update/interfaces.go; do
	if [ ! -f "$repo_root/$update_file" ]; then
		printf 'missing update root file: %s\n' "$update_file" >&2
		exit 1
	fi
done
test ! -e "$repo_root/internal/update/facade.go"
test ! -e "$repo_root/internal/update/types.go"

if ! (CDPATH= cd -- "$repo_root" && go list ./... >/dev/null); then
	printf '%s\n' 'Go package graph cannot be loaded without import cycles or other load errors' >&2
	exit 1
fi

# internal/common 根目录不建立 Go package。
if find "$repo_root/internal/common" -maxdepth 1 -name '*.go' | rg -q .; then
	printf '%s\n' 'internal/common root must not contain Go files' >&2
	exit 1
fi

# common 只允许基础转换，不得反向依赖 CLI、SDK、App API、config 或 update。
if rg -n 'github.com/FlanChanXwO/javdb-cli/(internal/cli|sdk|internal/javdb|internal/config|internal/update)' \
	"$repo_root/internal/common" -g '*.go'; then
	printf '%s\n' 'internal/common imports an upper-layer package' >&2
	exit 1
fi

# result 是纯投影层，只允许 stdlib 与 internal/common/scalar。
if rg -n 'github.com/FlanChanXwO/javdb-cli/(internal/cli/(commands|client|authstore|entity|invocation)|sdk|internal/javdb|internal/config|internal/storage|internal/update)' \
	"$repo_root/internal/cli/result" -g '*.go'; then
	printf '%s\n' 'internal/cli/result imports an upper-layer package' >&2
	exit 1
fi

# invocation 只依赖 stdlib IO。
if rg -n 'github.com/FlanChanXwO/javdb-cli/' \
	"$repo_root/internal/cli/invocation" -g '*.go'; then
	printf '%s\n' 'internal/cli/invocation imports a repository package' >&2
	exit 1
fi

if rg -n -F 'github.com/FlanChanXwO/javdb-cli/javdb' \
	"$repo_root/cmd" \
	"$repo_root/internal" \
	"$repo_root/sdk" \
	-g '*.go'; then
	printf '%s\n' 'source code imports the retired public SDK path' >&2
	exit 1
fi

if rg -n 'internal/javdb/(appapi|protocol)' \
	"$repo_root/cmd" \
	"$repo_root/internal/cli" \
	-g '*.go'; then
	printf '%s\n' 'CLI or binary entry imports a JavDB protocol implementation directly' >&2
	exit 1
fi

if rg -n 'internal/javdb/(appapi/(client|model|codec|media|endpoint)|protocol)' \
	"$repo_root/sdk" \
	-g '*.go'; then
	printf '%s\n' 'SDK source imports an app-API implementation package instead of the root facade' >&2
	exit 1
fi

# 这些是基线已经存在的 facade/返回类型兼容依赖；新增 SDK internal 依赖必须先改变契约并更新门禁。
# reversesearch/provider 是公开反搜方法的 wire 实现，SDK 只做类型映射，不暴露 internal 类型；
# reversesearch/image 提供 CLI 与 SDK 共用的格式/大小校验入口。
sdk_internal_imports=$(rg -n 'github.com/FlanChanXwO/javdb-cli/internal/' "$repo_root/sdk" -g '*.go' || true)
if [ -n "$sdk_internal_imports" ] && printf '%s\n' "$sdk_internal_imports" | rg -n -v 'github.com/FlanChanXwO/javdb-cli/internal/(config|javdb/appapi|storage/tags|reversesearch/provider|reversesearch/image)(/|")'; then
	printf '%s\n' 'SDK imports an internal package outside the compatibility allowlist' >&2
	exit 1
fi

# 根 facade 之外的 App API 领域包不得反向依赖 facade，避免循环和业务回流。
if rg -n 'github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi"' \
	"$repo_root/internal/javdb/appapi/client" \
	"$repo_root/internal/javdb/appapi/model" \
	"$repo_root/internal/javdb/appapi/codec" \
	"$repo_root/internal/javdb/appapi/media" \
	"$repo_root/internal/javdb/appapi/endpoint" \
	-g '*.go'; then
	printf '%s\n' 'App API implementation packages import the compatibility facade' >&2
	exit 1
fi

if rg -n 'github.com/FlanChanXwO/javdb-cli/(internal/cli|sdk)' \
	"$repo_root/scripts/releasenotes" \
	"$repo_root/scripts/internal/releasenotes" \
	-g '*.go'; then
	printf '%s\n' 'release-note implementation imports the CLI or public SDK' >&2
	exit 1
fi

# SDK 保留已弃用的 API() 兼容入口；这里检查公开文档不得泄露 internal 类型名称。
sdk_doc=$(CDPATH= cd -- "$repo_root" && go doc -all github.com/FlanChanXwO/javdb-cli/sdk)
if printf '%s\n' "$sdk_doc" | rg -n 'internal/|internal\\.javdb|protocol/(httpx|signature)'; then
	printf '%s\n' 'public SDK documentation exposes an internal implementation type' >&2
	exit 1
fi
