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
javdb/                             # 公开 Go SDK facade
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

1. Quality workflow 在 PR 与 `main` 上运行格式、测试、vet、构建和静态脚本门禁。
2. Platform smoke workflow 在六个原生 runner 测试、打包、解包并执行 `javdb version --json`。
3. `vX.Y.Z` tag 必须不可变且可追溯到 `main`；Release workflow 先在六个平台验证 tag 源码，再从全新 runner 重建唯一允许发布的资产。
4. 发布器核对资产、创建 GitHub Release、从同一 `checksums.txt` 渲染 Homebrew Formula，并在 macOS/Linux 的 amd64/arm64 环境验证。
5. tap 部署是可选的：必须设置 `HOMEBREW_TAP_DEPLOY_ENABLED=true` 并在受保护 `release` environment 配置 `HOMEBREW_TAP_DEPLOY_KEY`；条件缺失时 Release 与 Formula 验证仍会完成。

改 workflow、目标矩阵、打包或 Formula 时，同步改脚本测试、README 安装说明和本页。
