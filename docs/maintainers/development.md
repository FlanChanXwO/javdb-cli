# 开发指南

本页是维护者的 canonical 开发流程。公开 CLI 和 SDK 文档分别位于
`docs/en/` 与 `docs/zh-CN/`。

## 环境与快速验证

- 使用 `go.mod` 声明的 Go 版本。
- 单元测试、race、vet、构建和文档/发布结构检查默认不需要真实 JavDB 凭据。
- 不安装缺失的系统依赖或执行带凭据的在线命令，除非用户明确授权。

```bash
go test ./...
go test -race ./...
go vet ./...
go test ./scripts/...
sh scripts/build.sh
sh scripts/test-releasenotes.sh
sh scripts/test-package-release.sh
sh scripts/test-homebrew-formula.sh
sh scripts/test-workflows.sh
sh scripts/test-documentation.sh
sh scripts/test-architecture.sh
```

`gopls check ./...` 在部分 gopls 版本会把 `./...` 当作文件路径；若命令行版本不接受
该参数，使用等价的全仓文件诊断：

```bash
rg --files -g '*.go' -0 | xargs -0 gopls check
```

若本机安装了 `pre-commit`，在交付前运行：

```bash
pre-commit run --all-files
```

构建产物是 `build/javdb`（Windows 为 `build/javdb.exe`）。可用
`./build/javdb version --json` 复核 linker metadata。

## 目录地图

```text
cmd/javdb/                              # 二进制入口 → cli.Run
sdk/                                    # 公开 Go SDK（package javdb）
internal/cli/root.go                    # 真实根命令、persistent flags、注册顺序与 Run
internal/cli/root_test.go               # 根目录唯一测试文件（CLI-wide 契约）
internal/cli/invocation/                # RootOptions + Streams（调用期数据，仅 stdlib）
internal/cli/client/                    # 配置解析、SDK client、required/optional 认证生命周期
internal/cli/authstore/                 # 默认认证文件路径/目录/store 打开
internal/cli/result/                    # 纯结果投影与过滤（movie/magnet/named 分文件）
internal/cli/entity/                    # 六实体查询用例 Execute
internal/common/{jsonx,scalar}/         # 纯 JSON 与标量转换（根目录无包）
internal/cli/commands/{auth,config}/    # 认证与配置命令域
internal/cli/commands/{search,detail,comments,magnets,download,tags,browse}/  # 影片目录命令域
internal/cli/commands/{actor,series,maker,director,code,list}/                # 六个实体命令域
internal/cli/commands/{watched,want,recent,collections,mark,unmark}/          # 个人状态命令域
internal/cli/commands/{rankings,top250}/ # 排行命令域（rankings 含 movies/actors/playback）
internal/cli/commands/{lists,update,version}/ # 列表、更新和版本命令域（update 拥有 proxy/coordinator/buildinfo）
internal/javdb/appapi/                  # 真实 Client 组合层（client.go）
internal/javdb/appapi/{client,model}/   # transport 与 wire/domain model
internal/javdb/appapi/endpoint/*        # auth/browse/entity/lists/magnets/movie/rankings/search/user
internal/javdb/appapi/{codec,media}/    # 解码与媒体传输
internal/javdb/protocol/httpx/          # TLS 指纹 HTTP transport
internal/javdb/protocol/signature/      # 请求签名协议
internal/config/{paths,settings}/       # 路径与配置 schema（根目录无包）
internal/storage/auth/                  # 多账号 auth.json（model/store/file/resolve）
internal/storage/tags/                  # 公开标签目录缓存（model/file/resolve）
internal/buildinfo/                     # linker 注入版本信息
internal/update/                        # Coordinator 与最小依赖接口（coordinator.go/interfaces.go）
internal/update/{model,archive}/        # 更新模型与归档校验/安装
internal/update/{release,source,process}/ # Release、来源和进程/平台边界
scripts/releasenotes/                   # release-note command 入口（仅分派）
scripts/internal/releasenotes/{model,github,audit,document,prepare,history}/ # release-note 领域实现
scripts/                                # 构建、打包和静态检查
skills/javdb-cli/                  # 面向产品使用者的 agent skill
.agents/skills/                    # 仓库 review/docs/commit/release skills
changelog/                         # 双语版本化发布说明与 release-prep plans
docs/en/, docs/zh-CN/              # 公开接口文档
docs/maintainers/                  # 维护者架构、流程、ADR 与协作规则
```

完整边界见 [架构说明](architecture.md)。新目录应按真实职责加入，不为与
pixiv-cli 对称而创建空 application、bootstrap、MCP 或下载层。

CLI 命令包只通过 `sdk/` 执行远程 JavDB 操作；`cli/client` 统一配置解析与 SDK client
创建，`cli/authstore` 打开默认认证 store，`cli/result` 提供纯结果投影，显式 `update`
依赖由 `commands/update` 组装。App API endpoint 只依赖自己的
`client/model/codec/media`、必要的其他 endpoint capability 与 storage taxonomy，协议 `httpx/signature`
不得被 CLI 命令包直接导入。`internal/cli`、`internal/javdb/appapi`、`internal/config`、
`internal/update` 的根文件不再保留兼容 facade/alias/forwarder；`internal/cli` 根包只保留
`root.go`/`root_test.go`。新增实现应放入真实职责子包，并以聚焦测试覆盖原有契约。
`Client.API()` 与 taxonomy `*tags.Doc` 返回类型是公开 SDK 的冻结兼容例外，新能力不得扩大。

## 本机状态与在线验证

| 路径 | 内容 |
| --- | --- |
| `~/.javdb-cli/auth.json` | 账号、密码和 JWT；支持 POSIX 权限的平台为 `0600`。 |
| `~/.javdb-cli/config.toml` | host、proxy、auto_relogin、lang。 |
| `~/.javdb-cli/device_uuid` | 稳定的公开 device UUID。 |
| `~/.javdb-cli/tags-*.json` | 公开标签目录缓存，不含密钥。 |

真实 API 抽查会使用本机账号且可能改变 token、写入 tag cache 或访问远程状态；它不是默认回归。
仅在用户明确授权、凭据来源清楚且不会输出 secret 时再运行。

## Release-note 工具

`scripts/releasenotes` 的入口保留 `validate`、`audit`、`prepare`、`render`、`pr-validate`
和 `sync-history` 六个子命令；业务实现按 `scripts/internal/releasenotes/` 下的
`model`、`github`、`audit`、`document`、`prepare`、`history` 分域。脚本的专门行为门禁为：

```bash
sh scripts/test-releasenotes.sh
```

`validate`、`render`、`audit` 和不带 `--apply` 的 `prepare`/`sync-history` 可以离线
检查或 dry-run；`prepare --apply` 会写入本地 changelog，`sync-history --apply` 会修改
GitHub Release 正文。后两者不是默认回归，不得在测试中使用真实仓库凭据或隐式写入。

## 构建、打包与平台

Release 只支持六个原生目标：`darwin/amd64`、`darwin/arm64`、`linux/amd64`、
`linux/arm64`、`windows/amd64`、`windows/arm64`。Release binary 使用
`CGO_ENABLED=0`、`-trimpath` 和 `-buildvcs=false`；每个 archive 只包含目标二进制、
`LICENSE` 与 `README.md`。

本地演练一个目标而不发布：

```bash
mkdir -p dist
sh scripts/build-release.sh \
  --version 0.2.0 \
  --target darwin/arm64 \
  --output dist/javdb
sh scripts/package-release.sh \
  --binary dist/javdb \
  --version 0.2.0 \
  --target darwin/arm64 \
  --output-dir dist
```

`package-release.sh` 会拒绝不支持的平台、错误二进制名、符号链接输出和既有资产名。
Windows Git Bash runner 用预装 `7z` 生成 ZIP。

`javdb update` 依赖 Release 中与当前目标严格匹配的 archive 及 `checksums.txt`。安装器在替换
二进制前必须验证该 archive 的 SHA-256，并执行候选二进制的 `version --json` 核对 tag；因此变更
资产命名、平台矩阵或 checksum 格式时，必须同步更新 `internal/update` 的测试和用户文档。

## CI 与发布

1. Quality workflow 在 PR 与 `main` 上运行 release-note metadata 校验。仅限 README、贡献指南、
   changelog、docs、skills、repo-local skills 和 Issue/PR template 的改动走文档门禁；其他任何路径
   都运行格式、测试、vet、构建和静态脚本门禁。
2. Platform smoke workflow 用同一分类器在六个原生 runner 测试、打包、解包并执行
   `javdb version --json`；`Platform smoke gate` 始终存在，文档改动时显式确认矩阵被跳过。
3. feature PR 只填写 `.github/PULL_REQUEST_TEMPLATE.md` 中唯一的 release-note declaration；不要
   编辑 `changelog/unreleased/`。release-prep PR 将审核后的双语计划放入
   `changelog/plans/vX.Y.Z.json`，然后运行 `prepare` 与 `validate` 生成版本目录。
4. `vX.Y.Z` tag 必须不可变且可追溯到 `main`；Release workflow 在打包前校验版本化双语 notes、
   审计来源，并以同一渲染正文创建 GitHub Release。
5. 发布器核对资产、从同一 `checksums.txt` 渲染 Homebrew Formula，并在 macOS/Linux 的 amd64/arm64 环境验证。tap 部署是可选的：必须设置 `HOMEBREW_TAP_DEPLOY_ENABLED=true` 并在受保护 `release` environment 配置 `HOMEBREW_TAP_DEPLOY_KEY`；条件缺失时 Release 与 Formula 验证仍会完成。

历史 Release 回写、GitHub description 修改、release-prep PR 合并、创建 tag 与发布均是外部写入。
在当前会话取得目标版本、范围和影响的明确授权后，先 dry-run，再使用 `sync-history --apply` 或
对应 GitHub 命令。本机 `audit` 与 `sync-history` dry-run 只需要仓库和 PR 读取权限的 `GH_TOKEN`；
只有 `sync-history --apply` 及上述外部写入才需要相应写权限。GitHub Actions 审计使用受限的
只读 `github.token`。不得创建额外 tag、修改已发布资产或编造 PR 来源。

改 workflow、目标矩阵、打包或 Formula 时，同步改脚本测试、README 安装说明和本页。
