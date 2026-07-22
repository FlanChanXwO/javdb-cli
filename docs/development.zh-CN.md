# 开发指南

[English](development.md) | [简体中文](development.zh-CN.md)

本地构建与测试 **javdb**。

## 环境要求

- Go 版本见 `go.mod`（1.26.x）
- 仅可选真机冒烟需要网络（`go test` 不需要）

## 构建

```bash
sh scripts/build.sh
# 可选写入版本：
VERSION=0.1.0 sh scripts/build.sh
./build/javdb version --json
```

`scripts/build.sh` 会注入：

```text
-X …/internal/buildinfo.Version
-X …/internal/buildinfo.Commit
-X …/internal/buildinfo.BuildDate
```

发版 CI 应使用相同 ldflags，并设置 `CGO_ENABLED=0`。
本地产物为 `build/javdb`（Windows 为 `build/javdb.exe`）。

## 测试

```bash
go test ./...
go test -race ./...
go vet ./...
sh scripts/test-package-release.sh
sh scripts/test-homebrew-formula.sh
sh scripts/test-workflows.sh
```

单测默认离线（参数构造、输出、账号存储等）。  
不要提交凭证。真机抽查示例：

```bash
javdb auth login   # 一次即可
javdb search SSIS-589 --limit 1
javdb detail SSIS-589
javdb magnets SSIS-589 --best
```

## 目录结构

```text
cmd/javdb/main.go       # 入口 → cli.Run
javdb/                  # 公开 SDK
internal/appapi/        # HTTP 客户端与端点
internal/cli/           # Cobra 命令与输出
internal/config/        # 路径、config.toml 合并
internal/httpx/         # TLS 指纹 HTTP 客户端
internal/signature/     # 请求签名头
internal/storage/auth/  # 多账号 auth.json
internal/storage/tags/  # 标签分类文件
internal/buildinfo/     # 版本元数据
scripts/build.sh
scripts/build-release.sh
scripts/package-release.sh
skills/javdb-cli/       # 面向 agent 的操作 skill 与专题参考
docs/                   # 产品文档（中英）
```

## 主机

| 名称 | 基址 |
|------|------|
| `mirror`（默认） | `https://jdforrepam.com` |
| `main` | `https://javdb.com` |

优先用镜像直连；访问主站时可加 `--proxy`。

## 配置文件

| 路径 | 说明 |
|------|------|
| `~/.javdb-cli/auth.json` | 账号与 default_user_id；支持 POSIX 权限的平台为 `0600` |
| `~/.javdb-cli/config.toml` | host、proxy、auto_relogin、lang |
| `~/.javdb-cli/device_uuid` | 稳定的 device id（公共参数） |
| `~/.javdb-cli/tags-*.json` | 标签目录缓存（非密钥） |

## 风格约定

- CLI 参数名保持稳定，方便脚本与 agent。
- 纯函数（掩码、过滤）优先，配表驱动测试。
- 鉴权失败不得打印 token / 密码。
- `docs/` 只写产品文档（不写逆向过程）。

## 发版与平台验证

发版契约固定覆盖六个原生目标：`darwin/amd64`、`darwin/arm64`、
`linux/amd64`、`linux/arm64`、`windows/amd64`、`windows/arm64`。发布二进制使用
`CGO_ENABLED=0`、`-trimpath` 和 `-buildvcs=false` 构建；每个归档只含目标二进制、
`LICENSE` 与 `README.md`。

可在本地演练单一目标（不会发布）：

```bash
mkdir -p dist
sh scripts/build-release.sh \
  --version 0.1.1 \
  --target darwin/arm64 \
  --output dist/javdb
sh scripts/package-release.sh \
  --binary dist/javdb \
  --version 0.1.1 \
  --target darwin/arm64 \
  --output-dir dist
tar -xzf dist/javdb-cli_0.1.1_darwin_arm64.tar.gz -C /tmp/javdb-smoke
/tmp/javdb-smoke/javdb version --json
```

`package-release.sh` 会拒绝不支持的平台、错误二进制名、符号链接输出路径和已存在的
资产名。Windows Git Bash runner 没有 `zip`，脚本会明确使用该 runner 预装的 `7z`。

GitHub Actions 参照 pixiv-cli 的发版分段，但按本项目纯 Go 特性收敛：

1. **Quality gate** 在每个 PR 与 `main` push 上运行格式化、单测/race、vet、本地构建，以及打包/workflow 测试。
2. **Platform packaged binary smoke** 在六个原生 runner 上测试、构建、打包、解包，并执行 `javdb version --json`。
3. `vX.Y.Z` tag 必须不可变且可追溯至 `main`。六个原生 job 先验证 tag 源码，随后由全新 job 重建唯一允许发布的资产。
4. 发布 job 先创建 draft Release，再比对上传资产与本地已验证集合；发布后用同一份 `checksums.txt` 渲染 Homebrew Formula，并在 macOS/Linux runner 安装和校验 staging Formula。
5. 推送 tap 是显式可选项：设置仓库变量 `HOMEBREW_TAP_DEPLOY_ENABLED=true`，并在受保护的 `release` environment 中配置 `HOMEBREW_TAP_DEPLOY_KEY`。两者缺失时 Release 与 Formula 验证仍完成，但部署 job 会跳过。
