# 架构说明

## 总体流程

`cmd/javdb/main.go` 是唯一官方二进制入口，只负责把进程参数和标准流交给
`internal/cli.Run` 并返回退出码。`internal/cli` 的根包（`root.go`）直接组装最终
Cobra 命令树并实现 `Run`；命令域位于 `internal/cli/commands/*`（每个真实命令一个
同名目录），调用期数据由 `cli/invocation` 提供，配置/SDK client/认证生命周期由
`cli/client` 提供，认证文件由 `cli/authstore` 提供，纯结果投影由 `cli/result` 提供，
六实体查询用例由 `cli/entity` 提供。远程 JavDB 操作只能通过公开 `sdk/`（声明为
`package javdb`）调用；`sdk` 再组合 `internal/javdb/appapi` 的真实 Client 组合层与
协议实现。

```text
cmd/javdb → internal/cli (root.go → commands/*) → sdk (public; package javdb)
                         ├── cli/invocation              （RootOptions + Streams）
                         ├── cli/client → config/{paths,settings} + storage/route + sdk
                         ├── cli/authstore → storage/auth
                         ├── cli/result                   （movie/magnet/named 纯投影）
                         ├── cli/entity                   （六实体查询用例）
                         └── internal/common/{jsonx,scalar}（纯 JSON/标量）
sdk → internal/javdb/appapi (真实 Client 组合 → client/model/codec/media/endpoint/*)
                              └── internal/javdb/protocol/{httpx,signature}
internal/javdb/appapi → internal/storage/tags (taxonomy cache)
```

`config`、`storage` 与 `update` 负责本机状态或显式更新，而不是终端输出或远程协议。
CLI 可以通过 `cli/client`、`cli/authstore` 和对应命令域使用它们完成账号、配置和
update 命令；CLI 命令包不应直接导入 `internal/javdb/appapi` 或
`internal/javdb/protocol/*`。

## 包职责

### `cmd/javdb`

二进制 `main` package。不得承载命令逻辑、配置读取或 API 构造。

### `internal/cli`

命令适配层：Cobra 命令树、flag/参数校验、交互提示、文本或 JSON 输出，以及
将用户选择映射为 `javdb` 的公开请求类型。它维护 CLI 输出兼容性，但不实现
HTTP、签名或上游响应解码。目录职责如下：

- `cli/root.go`：persistent flags、根命令、26 个命令的注册顺序和 `Run` 入口（含
  Windows 待清理更新）；`internal/cli` 根包只保留 `root.go` 与 `root_test.go`，
  没有渲染/输入 wrapper 或命令专属测试。
- `cli/invocation`：`RootOptions`（persistent flags 目标）与 `Streams`（调用方提供的
  标准流），只保存调用期数据，不含服务方法。
- `cli/client`：统一配置解析（config path/file/host 校验/runtime 解析/device UUID）、
  SDK client 创建，以及 `WithRequiredAuth`/`WithOptionalAuth` 的认证生命周期（自动
  重登、匿名重试、token 持久化）。对默认 `auto` host，构造业务 client 前先读
  `storage/route` cache，用公开 SDK `SelectAutoHost` 验证/重选线路，必要时持久化新主机；
  固定 host 完全绕过 cache 与 selector。
- `cli/authstore`：只负责默认认证文件的路径、目录与 store 打开（`Open`）。
- `cli/result`：纯结果投影与过滤，按领域分文件（`movie.go`/`magnet.go`/`named.go`），
  类型与函数使用领域前缀（`MovieRow`/`ProjectMovie`/`FilterMoviesWithMagnets`、
  `MagnetRow`/`ProjectMagnet`、`NamedRow`/`ProjectNamed`）。
- `cli/entity`：只保留六类实体命令共享的查询用例 `Execute`；命名实体投影位于
  `cli/result`。
- `cli/commands/{auth,config,search,detail,comments,magnets,download,tags,browse,actor,series,maker,director,code,list,watched,want,recent,collections,mark,unmark,rankings,top250,lists,update,version}`：
  每个目录对应一个真实命令或命令组，主文件与目录同名；每个命令持有自己的 Cobra
  metadata、参数校验、flag、文本和 JSON 写入；远程操作只通过 `sdk`。
  `commands/update` 同时拥有独立于 JavDB host 设置的 proxy 解析、production coordinator 组装与 build info
  获取（未导出 helper）。

### `sdk/`（`package javdb`）

公开 Go SDK，导入路径为 `github.com/FlanChanXwO/javdb-cli/sdk`，声明为
`package javdb`。它提供 client options、稳定的操作方法、公开的请求/错误别名、本机
device UUID helper、排行参数 helper、显式自动选线 `SelectAutoHost`，以及影片单页评论和
选定媒体下载的请求类型。CLI 与外部 Go 调用方应共享这条能力面；`internal` 下的包不是外部
集成 API。`SelectAutoHost` 显式联网选线并返回具体 URL，`javdb.New(WithHost("auto"))`
不会自动联网。排行 zone 与 period 的协议归一化由 `internal/javdb/appapi` 负责；`sdk`
暴露通用 `RankingPeriod`，并保留 `ActorPeriod` 废弃别名以兼容既有调用方，CLI 不预先复制
这套映射。

### `internal/javdb/appapi`

根包是**真实 Client 组合层**（`client.go`），不是手写转发 facade：`New` 按固定顺序
构造一次 transport 与各 endpoint capability，通过未导出指针类型别名嵌入
`Client`，用 method promotion 提供公开 SDK `Client.API()` 当前可见的扁平方法集；
不暴露可访问的 endpoint 字段，不保留一行式 forwarder。实现按职责位于以下子包：

- `appapi/client`：HTTP、签名 header、公共参数、token/device/lang、envelope、状态码和认证错误映射；它是唯一持有协议 transport 的 App API 子包。
- `appapi/model`：Options、SearchResult、错误类型及 wire/domain model。
- `appapi/endpoint/{auth,browse,entity,lists,magnets,movie,rankings,route,search,user}`：有状态 capability service；`endpoint/magnets` 保持纯 helper，`endpoint/route` 是自动选线 capability（startup 域名解密、并发探测与确定性选择），经根 Client 组合。
- `appapi/codec`：App JSON、JWT、用户 ID 和响应数组解析。
- `appapi/media`：图片格式校验/XOR 还原、HLS playlist/key/IV/PKCS#7 处理和独占文件写入，通过 fetch callback 接入 client。

详情给出的缩略图、首张预览图和已结束的单媒体 HLS 仍由 adapter 负责下载、解密并合并。
App API 不解析终端参数，也不格式化面向用户的输出。

### `internal/javdb/protocol/httpx` 与 `signature`

协议细节。`httpx` 构造 TLS 指纹 HTTP transport，供签名 API 和直接媒体传输复用；
`signature` 生成 API 请求所需签名头。两者只服务 JavDB adapter，不应被 CLI 或公开 SDK
调用方直接依赖。

### `internal/config`

根目录不建立 Go package；`config/paths` 负责配置目录、config/route/device/tag 路径，
并以同目录临时文件写入、`Sync`、关闭后 no-replace 原子发布、私有权限和失败清理安全创建
首次基线配置（并发调用方只会看到完整文件，且不覆盖已有配置），
`config/settings` 负责 TOML schema、默认值（`host` 缺省为 `auto`）、环境变量和运行时
合并。调用方直接依赖两个子包。配置优先级必须维持为命令行 flag > 环境变量 > 文件 > 默认值。

### `internal/common/jsonx` 与 `internal/common/scalar`

纯底层转换，根目录不建立 package：`jsonx` 提供 `ObjectArray`/`ObjectSlice`/
`RawString`/`MarshalLine`（`MarshalLine` 保证 SetEscapeHTML(false) 且恰好一个尾随
换行），`scalar` 提供 `String`/`Int64`。两个包不接收 `io.Writer`、不写输出、不含 CLI
文案、不吞编码错误，也不反向依赖 CLI/SDK/App API/config/update。CLI 浮点截断、
App API 前缀数字解析、各领域 truthy 规则、CLI 文案、密码输入、HLS、分页和错误降级
必须留在对应领域，不在此目录继续堆叠通用 helper。

### `internal/storage/auth`、`internal/storage/tags` 与 `internal/storage/route`

分别保存多账号认证状态、公开标签目录缓存与自动选线路由缓存。`auth` 按 model/store/
file/resolve 拆分，负责 JSON、临时文件替换、权限和默认账户；`tags` 按 model/file/resolve
拆分，负责 taxonomy 格式、缓存路径、权限和自由格式 tag 解析；`storage/route` 负责
`route.json` 的严格 JSON load、URL 校验与同目录临时文件 + Sync + 原子替换写入（`0600`），
只保存已验证的 host URL，不保存 proxy/token/时间戳/测速历史。认证数据包含密码和
JWT，任何调用路径都不得将其输出到日志、错误、JSON 或文档示例中。

### `internal/buildinfo`

保存 linker 注入的版本、提交和构建时间。开发构建保留明确的默认值，不伪造
发布版本。

### `internal/reversesearch`

独立反搜 domain，分为四层：`image`（本地路径/URL/stdin 的原始图片读取，
JPEG/PNG/WEBP magic 与 8 MiB 流式边界，错误分 input/download/status/
format/size/cancel stage）、`provider`（内置 AVScan 与声明式外部 HTTP
adapter，统一 multipart `file` 响应协议、三次总请求与 30/60s 退避）、
`cache`（本机文件缓存，source + 原图 SHA-256 key、30 天 TTL、原子 0600、
损坏显错）、以及 SDK 层的严格番号联动（`ResolveMovieIDExact`）。CLI 负责
解析 `config.toml` 的 `[reverse_search]`（含 `${ENV:NAME}` header 展开与
脱敏）并以 `javdb.ReverseSearchCache` 接口注入本机缓存；SDK 不读取
`~/.javdb-cli`。

### `internal/cli/pipeline`

`javdb.pipeline/v1` 机器协议核心：typed envelope（schema/kind/ref/id/data/
meta）、严格 JSONL 与逐行文本解码、输入分类（图片 magic → JSONL → 文本）、
输出模式（TTY 文本 / 非 TTY JSONL / 显式 --jsonl/--text/--json 互斥，单/批
JSON cardinality）、批处理执行（顺序保持、原位错误信封、最终非零），以及
Consumer/BatchRunner/ListProducer 三件套把只读/状态命令统一接入。命令不
复制解析逻辑；`auth login` 与密码提示排除通用 stdin。

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
- 改 CLI/common/App API/update/config/storage/release-note 的目录边界：同步本页、`development.md` 和 `scripts/test-architecture.sh`；只有用户可见契约改变时才更新 public docs、README 或 changelog。
