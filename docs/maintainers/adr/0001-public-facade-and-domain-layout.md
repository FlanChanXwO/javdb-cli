# ADR 0001: 公开 facade 与 JavDB 领域目录

## 状态

已采纳。

## 背景

项目同时提供 `javdb` Go SDK 和 `javdb` CLI。早期 API client、HTTP transport 与签名包直接位于
`internal/` 根目录，CLI 也直接导入 API adapter。这使公开 SDK、终端适配和协议实现的边界不清晰，
与 pixiv-cli 的开发者难以快速定位对应层。

## 决策

- 保留顶层 `javdb/` 作为唯一公开 Go SDK facade。
- 将 JavDB 专属实现收拢到 `internal/javdb/appapi` 与 `internal/javdb/protocol/*`。
- CLI 通过公开 facade 调用远程能力；本机配置与账号存储仍由各自的 `internal/config`、`internal/storage/*` 管理。
- 不为尚不存在的 MCP、下载或 application/bootstrap 用例创建空目录。

## 后果

外部 SDK 导入路径和 CLI 命令保持不变；内部 import path 改为显式的 JavDB domain path。新增远程
能力应先判断是否需要公开 facade 契约，再实现协议 adapter 与 CLI 呈现。以后若出现可复用应用用例，
再以真实职责为依据抽取 `internal/application` 或 `internal/bootstrap`。
