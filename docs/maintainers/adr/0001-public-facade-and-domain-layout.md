# ADR 0001: 公开 SDK 与 JavDB 领域目录

## 状态

已采纳。

## 背景

项目同时提供 `sdk/` 下声明为 `package javdb` 的 Go SDK 和 `javdb` CLI。早期 API client、
HTTP transport 与签名包直接位于 `internal/` 根目录，CLI 也直接导入 API adapter。这使公开
SDK、终端适配和协议实现的边界不清晰，与 pixiv-cli 的开发者难以快速定位对应层；配置、更新
和 release-note 工具也缺少按职责查找的目录边界。第二阶段（domainization）在收敛内部结构时
移除了为迁移建立的内部 facade/compat，把 App API 根包重写为真实 Client 组合层。

## 决策

- 保留顶层 `sdk/`（`package javdb`）作为唯一公开 Go SDK；不存在另一个顶层 `javdb/` SDK 目录。
- 将 JavDB 专属实现收拢到 `internal/javdb/appapi` 与 `internal/javdb/protocol/*`，其中 App API 按 `client`、`model`、`codec`、`media`、`endpoint/*` 分域；**根包是真实 Client 组合层**（`client.go`），通过未导出指针别名嵌入 capability 并用 method promotion 提供扁平方法集，不保留手写 forwarder。
- CLI 通过公开 SDK 调用远程能力；`internal/cli` 根包只保留 `root.go`/`root_test.go`（`New`/`Run` 与 CLI-wide 契约测试），命令域按 `commands/*` 的真实命令/命令组同名目录分域，调用期数据由 `cli/invocation`（`RootOptions`+`Streams`）提供，配置/SDK client/认证生命周期由 `cli/client` 提供，认证文件由 `cli/authstore` 提供，纯结果投影由 `cli/result`（movie/magnet/named 分文件、领域前缀类型）提供，六实体查询用例由 `cli/entity` 提供；不再有 `app`、`movie`、`magnet` 或根 facade。
- 本机配置、账号存储和显式更新分别由 `internal/config/{paths,settings}`、`internal/storage/{auth,tags}` 和 `internal/update/{model,archive,release,source,process}` 管理，根目录不保留 facade/alias/forwarder；`internal/config` 根目录不建立 Go package，`internal/update` 根包只保留 `Coordinator` 与其最小依赖接口。
- 纯 JSON/标量转换由 `internal/common/{jsonx,scalar}` 提供（根目录无包）；`internal/shared/values` 已删除。
- release-note command 保留 `scripts/releasenotes` 入口（仅分派），核心逻辑按 `scripts/internal/releasenotes/{model,github,audit,document,prepare,history}` 分域；`changescope` 和 `login_probe.go` 保持单一职责。
- 不为尚不存在的 MCP、下载或 application/bootstrap 用例创建空目录。

## 后果

外部 SDK 导入路径和 CLI 命令保持不变；内部 import path 改为显式的 JavDB domain path。
内部兼容 facade/compat 是已删除的过渡产物：CLI/App API/config/update/release-note 的根文件
不再承载 alias/forwarder；新增远程能力应先判断是否需要公开 SDK 契约，再实现 endpoint capability
与 CLI 命令。`Client.API()` 作为既有调用方的 deprecated 兼容入口保留（它返回 appapi 根
`*Client`，因此该根类型必须是真实组合层而非转发 facade）；taxonomy 两个方法的 `*tags.Doc`
返回类型也作为同一基线的冻结兼容例外，但新 SDK 方法不得继续扩大 internal 类型或 adapter
暴露面。以后若出现可复用应用用例，再以真实职责为依据抽取 `internal/application` 或
`internal/bootstrap`。

## 依赖约束

- `cmd/javdb` 只调用 `internal/cli.Run`。
- CLI 命令包只通过 `sdk/` 执行远程 JavDB 操作，不直接导入 `internal/javdb/appapi` 或 `internal/javdb/protocol/*`。
- `internal/cli` 根包只保留 `root.go`/`root_test.go`；`cli/client` 统一配置解析与 SDK client/认证生命周期，`cli/authstore` 打开认证 store，`cli/result` 只依赖 stdlib 与 `internal/common/scalar`，`cli/invocation` 只依赖 stdlib IO；不建立通用 service-locator 或 output 大包。
- App API 根 Client 只构造一次 transport 并按固定顺序组装 capability；endpoint 不得反向导入根包、SDK 或 CLI。
- `internal/javdb/protocol/httpx` 与 `signature` 只服务 App API；`internal/common/{jsonx,scalar}` 只提供底层转换，不能成为 UI/网络/错误处理 god package，也不得反向依赖 CLI/SDK/App API/config/update。
- `scripts/releasenotes` 只负责命令入口分派，领域逻辑由 `scripts/internal/releasenotes/*` 提供；这些脚本包不得反向依赖 CLI 或 SDK。
