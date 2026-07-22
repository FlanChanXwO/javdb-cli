# 架构说明

## 总体流程

`cmd/javdb/main.go` 是唯一官方二进制入口，只负责把进程参数和标准流交给
`internal/cli.Run` 并返回退出码。命令层处理 Cobra 参数、交互输入和文本/JSON
呈现；远程 JavDB 操作只能通过公开 `javdb` facade 调用。facade 再组合
`internal/javdb/` 下的 App API 与协议实现。

```text
cmd/javdb → internal/cli → javdb (public SDK) → internal/javdb/appapi
                                      ├── internal/config
                                      └── internal/storage/{auth,tags}
```

`config` 与 `storage` 负责本机状态，而不是远程业务协议。CLI 可以直接使用它们
完成账号和配置命令；CLI 不应再直接导入 `internal/javdb/appapi` 或其协议子包。

## 包职责

### `cmd/javdb`

二进制 `main` package。不得承载命令逻辑、配置读取或 API 构造。

### `internal/cli`

命令适配层：Cobra 命令树、flag/参数校验、交互提示、文本或 JSON 输出，以及
将用户选择映射为 `javdb` 的公开请求类型。它维护 CLI 输出兼容性，但不实现
HTTP、签名或上游响应解码。

### `javdb`

公开 Go SDK facade，导入路径为
`github.com/FlanChanXwO/javdb-cli/javdb`。它提供 client options、稳定的操作
方法、公开的请求/错误别名和本机 device UUID helper。CLI 与外部 Go 调用方应
共享这条能力面；`internal` 下的包不是外部集成 API。

### `internal/javdb/appapi`

签名 App JSON API adapter：端点、wire response、请求参数、认证状态和
本地标签分类刷新。它不解析终端参数，也不格式化面向用户的输出。

### `internal/javdb/protocol/httpx` 与 `signature`

协议细节。`httpx` 构造 TLS 指纹 HTTP transport；`signature` 生成请求所需
签名头。两者只服务 JavDB adapter，不应被 CLI 或公开 SDK 调用方直接依赖。

### `internal/config`

解析用户配置目录、`config.toml`、环境变量和命令行覆盖层。配置优先级必须维持
为命令行 flag > 环境变量 > 文件 > 默认值。

### `internal/storage/auth` 与 `internal/storage/tags`

分别保存多账号认证状态与公开标签目录缓存。认证数据包含密码和 JWT，任何
调用路径都不得将其输出到日志、错误、JSON 或文档示例中。

### `internal/buildinfo`

保存 linker 注入的版本、提交和构建时间。开发构建保留明确的默认值，不伪造
发布版本。

## 目录约定

目录与 pixiv-cli 采用相同的高层语义：`cmd/` 是入口，`internal/cli/` 是用户
适配层，顶层领域目录是公开 SDK，`internal/<domain>/` 是协议/领域实现，
`internal/storage/` 是本机持久化，`docs/maintainers/` 是开发者权威文档。JavDB
没有 MCP、下载器或 Rust 组件，不能为了目录对称创建空层。

## 修改路由

- 改 CLI 命令、flag、输出或配置语义：同步更新两个 locale 的 CLI reference、README 和 `skills/javdb-cli/`。
- 改公开 SDK：同步更新两个 locale 的 SDK 文档与架构说明。
- 改 API adapter 或协议行为：补充聚焦测试，并检查 facade 是否需要暴露相应契约。
- 改构建、资产、Homebrew 或 Release：同步更新开发指南、workflow 测试和 README 安装说明。
