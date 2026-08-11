# 架构说明

## 总体流程

`cmd/javdb/main.go` 是唯一官方二进制入口，只负责把进程参数和标准流交给
`internal/cli.Run` 并返回退出码。`internal/cli` 的根包（`root.go`）直接组装最终
Cobra 命令树并实现 `Run`；命令域位于 `internal/cli/commands/*`（每个真实命令一个
同名目录），共享投影由 `cli/{movie,magnet,entity}` 提供，IO/运行时/SDK client/认证/
update 依赖组装由 `cli/app` 提供。远程 JavDB 操作只能通过公开 `sdk/`（声明为
`package javdb`）调用；`sdk` 再组合 `internal/javdb/appapi` 的真实 Client 组合层与
协议实现。

```text
cmd/javdb → internal/cli (root.go → commands/*) → sdk (public; package javdb)
                         ├── cli/app → internal/config, internal/storage/auth, internal/update
                         ├── cli/movie | cli/magnet | cli/entity   （纯投影）
                         └── internal/common/{jsonx,scalar}       （纯 JSON/标量）
sdk → internal/javdb/appapi (真实 Client 组合 → client/model/codec/media/endpoint/*)
                              └── internal/javdb/protocol/{httpx,signature}
internal/javdb/appapi → internal/storage/tags (taxonomy cache)
```

`config`、`storage` 与 `update` 负责本机状态或显式更新，而不是终端输出或远程协议。
CLI 可以通过 `cli/app` 和对应命令域使用它们完成账号、配置和 update 命令；CLI 命令包
不应直接导入 `internal/javdb/appapi` 或 `internal/javdb/protocol/*`。

## 包职责

### `cmd/javdb`

二进制 `main` package。不得承载命令逻辑、配置读取或 API 构造。

### `internal/cli`

命令适配层：Cobra 命令树、flag/参数校验、交互提示、文本或 JSON 输出，以及
将用户选择映射为 `javdb` 的公开请求类型。它维护 CLI 输出兼容性，但不实现
HTTP、签名或上游响应解码。目录职责如下：

- `cli/root.go`：persistent flags、根命令、26 个命令的注册顺序和 `Run` 入口（含
  Windows 待清理更新）；`internal/cli` 根包不再有渲染/输入 wrapper。
- `cli/app`：IO、配置 runtime、SDK client、认证 store 和 update coordinator 的依赖组装。
- `cli/commands/{auth,config,search,detail,comments,magnets,download,tags,browse,actor,series,maker,director,code,list,watched,want,recent,collections,mark,unmark,rankings,top250,lists,update,version}`：
  每个目录对应一个真实命令或命令组，主文件与目录同名；每个命令持有自己的 Cobra
  metadata、参数校验、flag、文本和 JSON 写入；远程操作只通过 `sdk`。
- `cli/movie`：影片记录纯投影（`Row`/`Project`/`ProjectAll`/`FilterHasMagnets`）。
- `cli/magnet`：磁力记录纯投影（`Row`/`Project`/`ProjectAll`/`FormatSize`）。
- `cli/entity`：实体用例 `Execute` 与命名实体投影（`NamedRow`/`ProjectNamed`）。

### `sdk/`（`package javdb`）

公开 Go SDK，导入路径为 `github.com/FlanChanXwO/javdb-cli/sdk`，声明为
`package javdb`。它提供 client options、稳定的操作方法、公开的请求/错误别名、本机
device UUID helper、排行参数 helper，以及影片单页评论和选定媒体下载的请求类型。
CLI 与外部 Go 调用方应共享这条能力面；`internal` 下的包不是外部集成 API。
排行 zone 与 period 的协议归一化由 `internal/javdb/appapi` 负责；`sdk` 暴露通用
`RankingPeriod`，并保留 `ActorPeriod` 废弃别名以兼容既有调用方，CLI 不预先复制这套映射。

### `internal/javdb/appapi`

根包是**真实 Client 组合层**（`client.go`），不是手写转发 facade：`New` 按固定顺序
构造一次 transport 与各 endpoint capability，通过未导出指针类型别名嵌入
`Client`，用 method promotion 提供公开 SDK `Client.API()` 当前可见的扁平方法集；
不暴露可访问的 endpoint 字段，不保留一行式 forwarder。实现按职责位于以下子包：

- `appapi/client`：HTTP、签名 header、公共参数、token/device/lang、envelope、状态码和认证错误映射；它是唯一持有协议 transport 的 App API 子包。
- `appapi/model`：Options、SearchResult、错误类型及 wire/domain model。
- `appapi/endpoint/{auth,browse,entity,lists,magnets,movie,rankings,search,user}`：有状态 capability service；`endpoint/magnets` 保持纯 helper。
- `appapi/codec`：App JSON、JWT、用户 ID 和响应数组解析。
- `appapi/media`：图片格式校验/XOR 还原、HLS playlist/key/IV/PKCS#7 处理和独占文件写入，通过 fetch callback 接入 client。

详情给出的缩略图、首张预览图和已结束的单媒体 HLS 仍由 adapter 负责下载、解密并合并。
App API 不解析终端参数，也不格式化面向用户的输出。

### `internal/javdb/protocol/httpx` 与 `signature`

协议细节。`httpx` 构造 TLS 指纹 HTTP transport，供签名 API 和直接媒体传输复用；
`signature` 生成 API 请求所需签名头。两者只服务 JavDB adapter，不应被 CLI 或公开 SDK
调用方直接依赖。

### `internal/config`

根目录不建立 Go package；`config/paths` 负责配置目录、文件和 device/tag 路径，
`config/settings` 负责 TOML schema、默认值、环境变量和运行时合并。调用方直接依赖
两个子包。配置优先级必须维持为命令行 flag > 环境变量 > 文件 > 默认值。

### `internal/common/jsonx` 与 `internal/common/scalar`

纯底层转换，根目录不建立 package：`jsonx` 提供 `ObjectArray`/`ObjectSlice`/
`RawString`/`MarshalLine`（`MarshalLine` 保证 SetEscapeHTML(false) 且恰好一个尾随
换行），`scalar` 提供 `String`/`Int64`。两个包不接收 `io.Writer`、不写输出、不含 CLI
文案、不吞编码错误，也不反向依赖 CLI/SDK/App API/config/update。CLI 浮点截断、
App API 前缀数字解析、各领域 truthy 规则、CLI 文案、密码输入、HLS、分页和错误降级
必须留在对应领域，不在此目录继续堆叠通用 helper。

### `internal/storage/auth` 与 `internal/storage/tags`

分别保存多账号认证状态与公开标签目录缓存。`auth` 按 model/store/file/resolve
拆分，负责 JSON、临时文件替换、权限和默认账户；`tags` 按 model/file/resolve
拆分，负责 taxonomy 格式、缓存路径、权限和自由格式 tag 解析。认证数据包含密码和
JWT，任何调用路径都不得将其输出到日志、错误、JSON 或文档示例中。

### `internal/buildinfo`

保存 linker 注入的版本、提交和构建时间。开发构建保留明确的默认值，不伪造
发布版本。

### `internal/update`

根包只保留 `Coordinator`、其最小依赖接口（`interfaces.go`）与测试，不再 alias
子包类型或保留 forwarder；`Execute` 直接使用 `update/model` 的 `Request`/`Result`。
实现按职责位于 `update/model`（公开类型）、`update/release`（GitHub Release、SemVer
和 HTTP）、`update/source`（安装来源）、`update/archive`（archive/checksum/安装）与
`update/process`（命令执行、候选二进制校验、平台替换和路径解析）。`internal/cli`
只翻译 `javdb update` 的 flag 和输出，不直接下载资产或猜测包管理器归属。

### `scripts/releasenotes`

`scripts/releasenotes/main.go` 只保留命令入口和子命令分派（validate/audit/prepare/
render/pr-validate/sync-history）；核心实现位于
`scripts/internal/releasenotes/{model,github,audit,document,prepare,history}`：model
保存跨包数据模型，github 封装 REST 边界，audit 生成审计报告，document 负责 changelog
解析/渲染，prepare 负责计划生成，history 负责显式历史 Release 同步。根 command 不再
有 compat wrapper。`scripts/changescope` 与 `scripts/login_probe.go` 保持单一职责，
不为目录对称强行拆分；`login_probe.go` 经公开 SDK 构造 client 后再用 `API()`。

## 目录约定

目录与 pixiv-cli 采用相同的高层语义：`cmd/` 是入口，`internal/cli/` 是用户
适配层，顶层 `sdk/` 是公开 SDK（`package javdb`），`internal/<domain>/` 是协议/领域实现，
`internal/common/` 是纯共享转换，`internal/storage/` 是本机持久化，
`scripts/internal/releasenotes/` 是 release-note 工具的领域实现，
`docs/maintainers/` 是开发者权威文档。JavDB 没有 MCP、独立下载器
服务或 Rust 组件；媒体下载只属于 `sdk/` 与 `internal/javdb/appapi/media`，
不能为了目录对称创建空层。

内部兼容 facade/compat 是已经删除的既定过渡产物：`sdk/`、`internal/javdb/appapi`、
`internal/config` 和 `internal/update` 的根文件不再保留 alias/forwarder/兼容 seam。
`internal/javdb/appapi` 根 `Client` 保留唯一原因是公开 SDK `Client.API()` 返回该
类型；它必须是真实组合层。`sdk` 内部可以组合 `internal/config/settings`、
`internal/storage/tags` 和 App API 根 Client，但不得把 `internal/javdb/protocol/*`
暴露给 CLI 或外部调用方。`RefreshTagTaxonomy` 与 `LoadOrRefreshTaxonomy` 的返回类型
仍是 `*tags.Doc`，与 `Client.API()` 一样属于冻结的源码兼容例外；新 SDK 能力不得
再增加 internal 类型或 adapter 返回入口，必须使用 typed 方法。

## 修改路由

- 改 CLI 命令、flag、输出或配置语义：同步更新两个 locale 的 CLI reference、README 和 `skills/javdb-cli/`。
- 改公开 SDK：同步更新两个 locale 的 SDK 文档与架构说明。
- 改 API adapter 或协议行为：补充聚焦测试，并检查根 Client/`sdk` 是否需要暴露相应契约。
- 改构建、资产、Homebrew 或 Release：同步更新开发指南、workflow 测试和 README 安装说明。
- 改 CLI/common/App API/update/config/storage/release-note 的目录边界：同步本页、`development.md`、相关 ADR（若有）和 `scripts/test-architecture.sh`；只有用户可见契约改变时才更新 public docs、README 或 changelog。
