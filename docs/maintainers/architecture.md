# 架构说明

## 总体流程

`cmd/javdb/main.go` 是唯一官方二进制入口，只负责把进程参数和标准流交给
`internal/cli.Run` 并返回退出码。`internal/cli` 的根包（`root.go`）直接组装最终
Cobra 命令树并实现 `Run`；命令域位于 `internal/cli/*`（每个真实命令一个
同名目录），调用期数据由 `environment` 提供，配置/SDK client/认证生命周期由
`session` 提供，认证文件由 `session/authstore` 提供，纯结果投影由 `display` 提供，
六实体查询用例由 `target` 提供。远程 JavDB 操作只能通过公开 `sdk/`（声明为
`package javdb`）调用；`sdk` 再组合 `internal/javdb/appapi` 的真实 Client 组合层与
协议实现。

```text
cmd/javdb → internal/cli (root.go → commands/*) → sdk (public; package javdb)
                         ├── environment              （RootOptions + Streams）
                         ├── session → config/{paths,settings} + storage/route + sdk
                         ├── session/authstore → storage/auth
                         ├── display                   （movie/magnet/named 纯投影）
                         ├── target                   （六实体查询用例）
                         └── internal/common/{jsonx,scalar}（纯 JSON/标量）
sdk → internal/javdb/appapi (真实 Client 组合 → client/model/codec/media/endpoint/*)
                              └── internal/javdb/protocol/{httpx,signature}
internal/javdb/appapi → internal/storage/tags (taxonomy cache)
```

`config`、`storage` 与 `update` 负责本机状态或显式更新，而不是终端输出或远程协议。
CLI 可以通过 `session`、`session/authstore` 和对应命令域使用它们完成账号、配置和
update 命令；CLI 命令包不应直接导入 `internal/javdb/appapi` 或
`internal/javdb/protocol/*`。

## 包职责

### `cmd/javdb`

二进制 `main` package。不得承载命令逻辑、配置读取或 API 构造。

### `internal/cli`

CLI 只保留命令树、执行入口和依赖装配，以及各真实命令或命令组的目录。
`root.go` 定义根命令与全局 flag，`composition.go` 装配命令树，
`execution.go` 管理日志初始化和退出码，`help.go`
本地化 Cobra 帮助。`actor/series/maker/director/code/list` 分别保留自己的命令入口；
共享构造逻辑归 `internal/command`，远程操作统一经过公开 `sdk/`。

共享机制按模块级 owner 分工：

- `internal/environment`：标准流、终端状态和调用期诊断出口。
- `internal/session`：配置解析、选线、SDK 装配、默认账号及一次自动重登；
  `session/authstore` 打开本机账号存储。个人命令提前检查凭证，公开查询允许匿名。
  默认拒绝失效凭证并提示登录；显式开启 `auto_relogin` 后才重登一次。
  并发操作共享重登结果，token 成功持久化后重试失败操作。
- `internal/record`：`javdb.pipeline/v1`、管道输入、输出模式与有序批处理。
- `internal/display`：影片、磁力和实体的文本投影及展示。
- `internal/target`：实体解析与共享查询用例，通过 SDK 窄能力接口执行。
- `internal/logging`：调用期 `slog` logger，stderr text/JSON 格式、完整级别名和统一错误出口。
- `internal/i18n`：嵌入的 `en.json` / `zh-CN.json` 文案、系统语言选择与格式参数。
  条目按使用场景命名，完整消息与共享 flag 说明各自复用；协议值、命令示例和纯记录格式由代码持有。
  不改变 API 的 `lang` 配置；格式参数沿用 Go 规则，保留 `%w` 错误链。

单项失败只记录一次；批处理逐项记录并返回带失败数量的结果，根入口只输出多项汇总。
机器结果继续写 stdout，完整原始错误作为诊断字段保留，账号凭证和认证请求体保持私密。
图片反搜先取得 provider 候选，再通过公开 SDK `LinkImageCandidates` 和带认证的查询接口关联；
重登只重试被拒绝的 JavDB 操作，成功候选和 provider 上传保持原样。

### `sdk/`（`package javdb`）

公开 Go SDK，导入路径为 `github.com/FlanChanXwO/javdb-cli/sdk`，声明为
`package javdb`。它提供 client options、稳定的操作方法、`CheckAuth` 服务端认证检查、`LinkImageCandidates` 候选关联、公开的请求/错误别名、本机
device UUID helper、排行参数 helper、显式自动选线 `SelectAutoHost`，以及影片单页评论和
本地影片资源 API（`MovieAssets`、`DownloadMovieAsset` 与最小 `MovieAsset{Type,URL}` 模型，
辅以 `MovieAssetsFromDetail`、`MovieAssetDescriptions`、`ImageAssetFormat`）。CLI 与外部 Go 调用方应共享这条能力面；`internal` 下的包不是外部
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
- `appapi/media`：图片格式校验/XOR 还原、HLS playlist/key/IV/PKCS#7 处理、三层完整性
  （Layer A segment 校验、Layer B 媒体模型校验、Layer C MP4 容器校验）、按协议要求的
  segment 重试、spool 化 TS→MP4 Fast Start remux（含 segment 时间戳归一化）与 no-replace 发布，
  TS 临时文件逐包重组每个 PID 当前未完成的 PES，H.264/AAC 样本立即校验并写入
  spool；长期只保留 codec 配置与 sample metadata，最终 mdat 使用固定 I/O 缓冲复制。
  TS 发布前运行 Layer B；MP4 每段仅支持一个 H.264 PID、至多一个 AAC-LC PID，
  AAC 每 ADTS 帧须只有一个 raw data block，timed ID3 忽略。TS 保留模式不套用
  MP4 的 AAC profile/block 限制。
  峰值内存仍包含当前 PES 与 moov 元数据，不承诺与样本数无关的常量内存。
  通过 fetch callback 接入 client。assets 域（`javdb assets list|download`）不使用
  javdb.pipeline/v1 envelope，管道协议为 `TYPE<TAB>URL` 文本流。

  `assets list` 在 selector 完成后通过 SDK `ProbeMovieAssets` 执行默认启用的 bounded
  streaming metadata probe；固定 worker pool 只保留图片 header、HLS playlist 与首段 SPS
  所需的小量状态，不把完整媒体载荷放入内存。JSON/NDJSON 将 probe 结果作为可选字段，TTY
  展示尺寸/预览时长，pipe 仍严格保持 `TYPE<TAB>URL`。配置由 `[assets.probe]` 控制；默认配置
  文件保持 sparse，`config get` 展示 effective defaults。

详情给出的缩略图、首张预览图和已结束的单媒体 HLS 仍由 adapter 负责写入、解密并合并；
该能力不包含完整影片或磁力目标下载。
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

### `internal/common/{jsonx,scalar,atomicfile}`

纯底层转换与文件原语，根目录不建立 package：`jsonx` 提供 `ObjectArray`/`ObjectSlice`/
`RawString`/`MarshalLine`（`MarshalLine` 保证 SetEscapeHTML(false) 且恰好一个尾随
换行），`scalar` 提供 `String`/`Int64`。这些包不接收 `io.Writer`、不写输出、不含 CLI
文案、不吞编码错误，也不反向依赖 CLI/SDK/App API/config/update。CLI 浮点截断、
App API 前缀数字解析、各领域 truthy 规则、CLI 文案、密码输入、HLS、分页和错误降级
必须留在对应领域，不在此目录继续堆叠通用 helper。
`atomicfile.LinkNoReplace` 只提供同文件系统内的原子 no-replace 硬链接，供 CLI
图片自动命名与 media 的 TS/MP4 发布共享；它不负责目录创建、临时文件清理或输出文案。

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

### `internal/record`

`javdb.pipeline/v1` 机器协议核心：typed envelope（schema/kind/ref/id/data/
meta）、严格 NDJSON 与逐行文本解码、输入分类（图片 magic → NDJSON → 文本）、
输出模式（TTY 默认 OutputHuman 人类文本 / 非 TTY 默认 OutputText 稳定记录流 /
显式 --ndjson/--json 互斥，单/批 JSON cardinality）、批处理执行（顺序保持、
原位错误信封、最终非零；Concurrency > 0 时按输入并发并保序写出）、
Consumer/BatchRunner/ListProducer 三件套把只读/状态命令统一接入。RunMany
支持单项 fan-out（如 movie 搜索按影片展开）。Producer 可选的 RenderJSON
消费本次 Produce 已生成的 envelopes，保留命令既有的 legacy JSON shape 并避免
重复请求；未设置时继续使用 LegacyJSON fallback。命令不
复制解析逻辑；所有 `OutputJSON` / `OutputNDJSON` envelope 在序列化前必须通过
`Envelope.Validate()`，无效 schema、kind 或 ref/id 不得写出；`auth login` 与密码提示排除通用 stdin。

### `internal/update`

根包只保留 `Coordinator`、其最小依赖接口（`interfaces.go`）与测试，不再 alias
子包类型或保留 forwarder；`Execute` 直接使用 `update/model` 的 `Request`/`Result`。
实现按职责位于 `update/model`（公开类型）、`update/release`（GitHub Release、SemVer
和 HTTP）、`update/source`（安装来源）、`update/archive`（archive/checksum/安装）与
`update/process`（命令执行、候选二进制校验、平台替换和路径解析）。`internal/cli`
只翻译 `javdb update` 的 flag 和输出，不直接下载资产或猜测包管理器归属。

### `scripts/releasenotes`

`scripts/releasenotes/main.go` 只保留命令入口和子命令分派（validate/audit/render/
sync-history）；核心实现位于
`scripts/internal/releasenotes/{model,github,audit,document,history}`：model
保存跨包数据模型，github 封装 REST 边界，audit 生成审计报告，document 负责 changelog
解析/渲染，history 负责显式历史 Release 同步。版本化 changelog 是唯一发布说明来源，根 command 不再
有 compat wrapper。CI 变更范围由 `scripts/classify-change-scope.sh` 读取 `.github/ci-change-scope.gitignore` 判定；路径策略保持为数据文件，不建立 Go package。

## 目录约定

目录与 pixiv-cli 采用相同的高层语义：`cmd/` 是入口，`internal/cli/` 是用户
适配层，顶层 `sdk/` 是公开 SDK（`package javdb`），`internal/<domain>/` 是协议/领域实现，
`internal/common/` 是纯共享转换，`internal/storage/` 是本机持久化，
`scripts/internal/releasenotes/` 是 release-note 工具的领域实现，
`docs/maintainers/` 是开发者权威文档。JavDB 没有 MCP、独立下载器
服务或 Rust 组件；本地影片资源写入只属于 `sdk/` 与 `internal/javdb/appapi/media`，
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
- 改 CLI/common/App API/update/config/storage/release-note 的目录边界：同步本页和 `development.md`；只有用户可见契约改变时才更新 public docs、README 或 changelog。
