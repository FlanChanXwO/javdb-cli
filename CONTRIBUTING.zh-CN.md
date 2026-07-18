# 贡献指南

[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

感谢你参与改进 **javdb**。

## 开发环境

```bash
git clone https://github.com/FlanChanXwO/javdb-cli.git
cd javdb-cli
go test ./...
sh scripts/build.sh
./build/javdb version --json
```

- Go 版本：见 `go.mod`（当前 1.26.x）。
- **不要**提交 `~/.javdb-cli/auth.json`、token 或密码。
- 优先离线单测；可选真机冒烟使用本机凭证，且不得把密钥打进日志。

## 目录结构

```
cmd/javdb/          # 二进制入口
javdb/              # 公开 SDK
internal/
  appapi/           # 签名 App API 客户端
  cli/              # Cobra 命令与输出
  config/           # config.toml 路径与合并
  httpx/            # TLS 客户端封装
  signature/        # 请求签名
  storage/auth/     # 多账号存储
  storage/tags/     # 标签分类缓存
  buildinfo/        # 版本 ldflags
scripts/            # 构建脚本
docs/               # 用户与开发文档（中英）
```

## 编码约定

1. 有 Python 原型时，尽量对齐命令行为与参数名。
2. 文本与 JSON 输出保持稳定，方便 agent/脚本。
3. 新端点：先写参数构造单测（TDD）。
4. `docs/` 与 README 只写产品文档，不写逆向过程说明。
5. 鉴权错误信息要清晰；仅在配置开启时做 `auto_relogin`。

## Pull Request

1. 本地跑通 `go test ./...`、`go test -race ./...`、`go vet ./...`、`sh scripts/build.sh`。
2. 提交信息聚焦改动本身。
3. 用户可见变更请更新 `CHANGELOG.md` / `CHANGELOG.zh-CN.md`。
4. 启用 CI 后，`main` / PR 必须保持绿灯。

## 许可证

贡献代码即表示你同意以 [MIT License](./LICENSE) 授权你的贡献。
