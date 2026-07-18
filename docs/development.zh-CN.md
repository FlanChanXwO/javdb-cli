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

## 测试

```bash
go test ./...
go test -race ./...
go vet ./...
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
| `~/.javdb-cli/auth.json` | 账号与 default_user_id；权限 `0600` |
| `~/.javdb-cli/config.toml` | host、proxy、auto_relogin、lang |
| `~/.javdb-cli/device_uuid` | 稳定的 device id（公共参数） |
| `~/.javdb-cli/tags-*.json` | 标签目录缓存（非密钥） |

## 风格约定

- CLI 参数名保持稳定，方便脚本与 agent。
- 纯函数（掩码、过滤）优先，配表驱动测试。
- 鉴权失败不得打印 token / 密码。
- `docs/` 只写产品文档（不写逆向过程）。

## 发版清单（概要）

1. `go test` / race / vet / build 全绿  
2. 打 tag `vX.Y.Z`  
3. CI 产出多架构包与 checksums  
4. 更新 `FlanChanXwO/homebrew-tap` 中的 formula  

完整流水线见项目 plan / goal 任务列表。
