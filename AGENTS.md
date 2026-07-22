# AGENTS.md

`javdb-cli` 是 JavDB App JSON API 的 Go CLI 与公开 SDK。默认离线验证：

```bash
go test ./...
sh scripts/build.sh
```

## 架构边界

- `cmd/javdb` 只委托 `internal/cli`。
- `internal/cli` 负责 Cobra、输入和输出；远程 JavDB 操作只通过顶层 `javdb` public facade。
- `javdb` 是唯一公开 Go SDK；协议实现位于 `internal/javdb/appapi` 与 `internal/javdb/protocol/*`。
- `internal/config` 管理本机配置；`internal/storage/auth` 与 `internal/storage/tags` 管理本机状态。
- 不提交密码、JWT、`auth.json`、tag cache、构建产物或本机配置。

## 变更路由

- 命令、flag、JSON、配置或认证行为：更新两个 locale 的 CLI reference、README 与 `skills/javdb-cli/`。
- 公开 SDK：更新两个 locale 的 SDK 文档与 `docs/maintainers/architecture.md`。
- 构建或发布：更新 `docs/maintainers/development.md`、workflow 测试和 README。
- 用户可感知的变化：更新两个 changelog 的 `Unreleased`。

完整协作规则、审查清单与文档边界见 `docs/maintainers/agents/`；架构细节见
`docs/maintainers/architecture.md`。
