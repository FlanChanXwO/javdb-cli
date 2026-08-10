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
sh scripts/build.sh
sh scripts/test-package-release.sh
sh scripts/test-homebrew-formula.sh
sh scripts/test-workflows.sh
sh scripts/test-documentation.sh
sh scripts/test-architecture.sh
```

若本机安装了 `pre-commit`，在交付前运行：

```bash
pre-commit run --all-files
```

构建产物是 `build/javdb`（Windows 为 `build/javdb.exe`）。可用
`./build/javdb version --json` 复核 linker metadata。

## 目录地图

```text
cmd/javdb/                         # 二进制入口 → cli.Run
sdk/                               # 公开 Go SDK facade（package javdb）
internal/cli/                      # Cobra、交互和输出 adapter
internal/javdb/appapi/             # JavDB App JSON API adapter
internal/javdb/protocol/httpx/     # TLS 指纹 HTTP transport
internal/javdb/protocol/signature/ # 请求签名协议
internal/config/                   # 配置路径、文件和运行时合并
internal/storage/auth/             # 多账号 auth.json
internal/storage/tags/             # 公开标签目录缓存
internal/buildinfo/                # linker 注入版本信息
internal/update/                   # 显式更新、Release 校验与替换
scripts/                           # 构建、打包和静态检查
skills/javdb-cli/                  # 面向产品使用者的 agent skill
.agents/skills/                    # 仓库 review/docs/commit/release skills
changelog/                         # 双语版本化发布说明与 release-prep plans
docs/en/, docs/zh-CN/              # 公开接口文档
docs/maintainers/                  # 维护者架构、流程、ADR 与协作规则
```

完整边界见 [架构说明](architecture.md)。新目录应按真实职责加入，不为与
pixiv-cli 对称而创建空 application、bootstrap、MCP 或下载层。

## 本机状态与在线验证

| 路径 | 内容 |
| --- | --- |
| `~/.javdb-cli/auth.json` | 账号、密码和 JWT；支持 POSIX 权限的平台为 `0600`。 |
| `~/.javdb-cli/config.toml` | host、proxy、auto_relogin、lang。 |
| `~/.javdb-cli/device_uuid` | 稳定的公开 device UUID。 |
| `~/.javdb-cli/tags-*.json` | 公开标签目录缓存，不含密钥。 |

真实 API 抽查会使用本机账号且可能改变 token、写入 tag cache 或访问远程状态；它不是默认回归。
仅在用户明确授权、凭据来源清楚且不会输出 secret 时再运行。

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
6. `.github/CODEOWNERS` 将默认 review 路由到唯一维护者；它只会为未来 PR 请求 reviewer，不能让 PR 作者批准自己的 PR，也不替代 `main` 的分支保护要求。
7. Release 在公开 GitHub Release 后上传只含不可变 tag 的 `clawhub-release-tag` artifact。成功结束的
   `Release` workflow 会由 `publish-clawhub.yml` 通过 `workflow_run` 消费；它 checkout 该 tag、验证
   它属于默认分支，并跳过未改变的 `skills/javdb-cli/`。ClawHub 使用锁定的 `clawhub@0.23.1` 先做
   无凭据 dry-run，再在最后发布步骤读取 `CLAWHUB_TOKEN`。该 token 只应配置为仓库或受保护
   environment secret，绝不能写入 skill、workflow 日志或 Release notes。

历史 Release 回写、GitHub description 修改、release-prep PR 合并、创建 tag 与发布均是外部写入。
在当前会话取得目标版本、范围和影响的明确授权后，先 dry-run，再使用 `sync-history --apply` 或
对应 GitHub 命令。本机 `audit` 与 `sync-history` dry-run 只需要仓库和 PR 读取权限的 `GH_TOKEN`；
只有 `sync-history --apply` 及上述外部写入才需要相应写权限。GitHub Actions 审计使用受限的
只读 `github.token`。不得创建额外 tag、修改已发布资产或编造 PR 来源。

改 workflow、目标矩阵、打包或 Formula 时，同步改脚本测试、README 安装说明和本页。
