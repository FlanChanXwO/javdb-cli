# Goal 2 输入

## 用户请求

重整 `internal/cli` 的目录与职责，重点解决以下问题：

- `internal/cli` 根目录仍散落大量按命令命名的 `*_test.go`，测试 owner 不清晰。
- `internal/cli/app` 同时承担调用期数据、配置解析、SDK client、认证流程、update 组装和 build info，职责过多。
- `internal/cli/movie` 的名称过于具体，未表达它与 magnet/named 代码共同承担的结果投影职责。

本轮只把已经讨论并批准的方案写入 `goal-2/`，不实施业务代码、测试迁移、文档同步或架构门禁修改。

## 目标

以后续实施开始时的工作树为行为基线，在不改变公开 CLI 与 SDK 契约的前提下：

- 让 `internal/cli` 根包只负责 Cobra 根命令装配与进程级执行契约。
- 删除宽泛的 `app` 依赖汇聚包，按 invocation、client、auth store 等真实能力拆分。
- 用统一的 `result` 包承载 movie、magnet、named 的纯结果投影与过滤。
- 让命令专属测试回到 `commands/*` owner，根目录只保留 CLI-wide 契约测试。
- 保持命令包自己拥有参数校验、用户文案、文本/JSON IO 与错误语义，不建立新的通用 output/service-locator 包。

## 已确认决策

- 允许破坏 `internal` import path、内部导出符号和命令构造函数签名，不保留兼容 facade 或 alias。
- `internal/cli.New` 与 `internal/cli.Run` 的签名、根命令集合和外部行为保持不变。
- CLI 命令、子命令、flag、默认值、help、stdout、stderr、JSON 字节、退出码和网络前置校验保持不变。
- 公开 `sdk/` 的导出 API、错误匹配、`Client.API()` 与 taxonomy 返回类型保持不变。
- 删除 `internal/cli/app`、`internal/cli/movie`、`internal/cli/magnet`。
- 新建 `internal/cli/invocation`、`internal/cli/client`、`internal/cli/authstore`、`internal/cli/result`。
- `invocation` 只保存 `RootOptions` 与 `Streams`，不包含服务方法。
- `client` 统一配置解析、SDK client 创建及 required/optional auth 生命周期。
- `authstore` 只负责默认认证文件的路径、目录与 store 打开。
- update 的 proxy 解析、production coordinator 组装和 build info 获取归 `commands/update` 所有。
- `result` 是一个统一包，但按 `movie.go`、`magnet.go`、`named.go` 分文件；类型与函数使用领域前缀，避免含糊的 `Row`、`Project`。
- `entity` 只保留六类实体命令共享的查询用例，`NamedRow` 投影迁入 `result`。
- 不统一各命令重复的 writer；`result` 不接收 `io.Writer`，不含 Cobra、SDK 调用、JSON 编码或空列表文案。
- 根目录最终只保留 `root.go` 与一个 `root_test.go`。
- `goal-1/` 是历史实施记录，不回写其中已经确认的旧结构。
- 不自动 commit、建分支、push、发布或运行真实凭据、真实 JavDB API、在线 update。

## 不在本目标内

- 不新增、删除或重命名 CLI 命令、子命令或 flag。
- 不改变配置文件、认证文件、device UUID、tag cache 的路径、格式或权限。
- 不修改 HTTP、签名、App API endpoint 或公开 SDK 的实现边界。
- 不借本次重整统一命令文案、writer、JSON helper 或 Cobra 构造模式。
- 不新增 timeout、截断、重试上限、静默 fallback、错误吞咽或无依据的数据限制。
- 不更新 README、双语用户文档、产品 skill 或 changelog；若实施导致这些材料需要改变，应先视为公开契约偏离并修正实现。
- 不回退、覆盖或删除与本目标无关的用户工作树内容，包括 `.idea/`。

## 当前勘察基线

- 当前分支为 `codex/domainization-remove-facades`，勘察时 HEAD 为 `9e8f965dce1f7153db540338849bea8a0d944463`。
- 勘察时 `git status --short` 只有未跟踪的 `.idea/`；实施开始时必须重新记录实际状态，不以本记录覆盖新变化。
- `gopls v0.21.1` 可用，`internal/cli` 当前无诊断。
- `internal/cli` 当前包含 40 个测试文件；根目录有 13 个按命令或契约拆分的测试文件。
- 当前 `app` 有 3 个生产文件，覆盖 root options/streams、配置与 client、认证流程、update 组装和 build info。
- 当前 `movie.FilterHasMagnets` 有 9 个生产调用方；`entity.Execute` 有 6 个命令调用方；相关符号引用已用 `gopls references` 核对。
- 默认 `/usr/local/go/bin/go` 为 1.26.3，但 `GOROOT` 指向 1.26.4 模块工具链，直接运行会出现编译版本不匹配。
- 机器上现有的 Go 1.26.4 二进制已成功执行 `go test ./internal/cli/...`；实施验证必须显式使用该工具链，或在用户修复 shell 工具链后使用正常 `go`。

## 验收摘要

- 根目录只剩 `root.go` 与 `root_test.go`，命令专属测试均由对应包拥有。
- `app`、`movie`、`magnet` 路径及 import 全仓消失。
- 新能力包职责单一、依赖方向清晰，不形成新的 service locator 或 output 大包。
- 公开 CLI/SDK 与现有错误、认证、update 行为全部保持。
- 聚焦测试、全仓 test/race/vet/build、LSP、架构与文档门禁全部通过。
