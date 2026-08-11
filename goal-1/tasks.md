# Goal 1 任务清单

执行规则：

- 每轮完整读取 `input.md`、`plan.md`、`tasks.md`。
- 每轮只处理第一个 `pending` task，不合并、不跳序、不顺手执行相邻任务。
- 每三个普通 task 后执行一次集中检查-debug。
- 完成 task 前必须有 LSP、测试、构建或静态检查等可复现证据。
- 不自动 commit、建分支、push、发布或运行真实凭据/真实 API。
- 完成后填写“实际做了什么、验证证据、剩余风险、下一步”；本文件当前只初始化任务，不代表任何业务工作已经执行。

## Task 1：冻结当前工作树与行为基线

- 状态：`completed`
- 范围：记录实施开始时的 HEAD、origin 引用、dirty manifest、Go 包图、LSP、CLI 命令树、公开 SDK 符号和当前离线验证结果。
- 必须完成：
  - 记录 `git status --short`，区分现有用户文件、上一阶段迁移文件与本 task 新增基线测试。
  - 用 gopls references 和 `go list` 记录 CLI/config/update/App API facade 的真实调用方。
  - 记录 Cobra 顶层命令、子命令、AddCommand 顺序、persistent flags 和关键 help。
  - 记录 SDK `go doc -all`、`Client.API()` 方法集、taxonomy 返回类型和错误类型。
  - 运行当前离线基线检查，只记录现状，不修复与本目标无关的问题。
- 验证要求：gopls 当前诊断、`go list ./...`、`go test ./...`、`go vet ./...`、架构门禁和 `git diff --check`。
- 实际做了什么：
  - 引用：`HEAD=3da5d6f85051c951b29b6fdc084bb49e1780c140`（`v0.5.0-2-g3da5d6f`，当前分支 `codex/pr13-docs` 跟踪 `origin/pr-13`）；`origin/main=e84ad613d9867759b2aaeb3e50f6d02a933b2887`；历史参考 `9c3ee65`（skillhub-clawhub v0.5.2 merge，不再称为当前 origin/main）。三项均与 plan 观测值一致，无漂移；未 reset/checkout 恢复任何值。
  - Dirty manifest：`git status --short` 共 122 项（63 `D`：62 worktree + 1 staged `docs/superpowers/specs/2026-07-30-release-notes-system-design.md`；21 `M`；38 `??`）。
    - 用户既有文件（必须保留）：README.md、README.zh-CN.md、changelog/README.md、changelog/README.zh-CN.md、changelog/plans/v0.5.1.json、changelog/v0.5.1/、docs/{en,zh-CN}/cli-reference.md、docs/{en,zh-CN}/sdk.md、docs/maintainers/{adr/0001-public-facade-and-domain-layout,architecture,development}.md、sdk/rankings.go、skills/javdb-cli/SKILL.md、.idea/。
    - 上一阶段（initial-domainization，未提交）迁移文件：`internal/cli/{app,commands/{account,catalog,config,lists,rankings,update,user,version},input,output,root,facade.go,root.go}`；`internal/config/{facade.go,facade_test.go,paths/,settings/}`；`internal/javdb/appapi/{client,codec,endpoint/{auth,browse,entity,lists,magnets,movie,rankings,search,user},media,model,facade.go,facade_test.go}`；`internal/shared/values`；`internal/update/{archive,model,process,release,source,facade.go,facade_test.go}`；`internal/storage/{auth/{file,model,resolve}.go,store.go,store_test.go,tags/{file,model,resolve}.go,store_test.go}`；`scripts/internal/releasenotes/{audit,document,github,history,model,prepare}`；`scripts/releasenotes/{compat.go,main.go}`；`sdk/facade_test.go`；以及对应旧扁平文件删除（`internal/cli/*.go`、`internal/config/{paths,settings}.go`、`internal/javdb/appapi/*.go`、`internal/update/*.go`、`internal/storage/tags/store.go`、`scripts/releasenotes/{audit,history,prepare,render,validate}.go`、`scripts/internal/releasenotesrender/render.go`）。
    - 本 task 未新增测试/代码；契约基线测试属 Task 2。
  - Go 包图：`go list ./...` 52 个包可加载无循环；当前结构为旧 facade 与新领域子包并存（root facade：`internal/cli`、`internal/config`、`internal/update`、`internal/javdb/appapi`）。
  - Facade 真实调用方（rg import 检索；未用 gopls references 单独跑，因 import 文本已覆盖全部调用点）：
    - `internal/config` 根：`internal/cli/app/{app,update}.go`、`internal/cli/commands/config/config.go`、`internal/cli/facade.go`、`internal/storage/tags/file.go`、`sdk/client.go`。
    - `internal/update` 根：`internal/cli/app/update.go`、`internal/cli/commands/update/update.go`、`internal/cli/output/update.go`、`internal/cli/root/root.go`。
    - `internal/javdb/appapi` 根：`scripts/login_probe.go`、`sdk/{browse,client,entity,errors,facade_test,magnets,rankings,search}.go`。
    - `internal/cli/root`：仅 `internal/cli/root.go`（root facade 转发）。
    - `internal/cli/app`：`internal/cli/commands/*` 全部命令 + `internal/cli/facade.go` + `internal/cli/root/root.go`。
    - `internal/cli/output`：`internal/cli/commands/*` + `internal/cli/facade.go`。
    - `internal/shared/values`：`internal/cli/output/print.go`、`internal/javdb/appapi/{codec, endpoint/browse, endpoint/entity, endpoint/magnets, endpoint/movie/resolve, endpoint/user}.go`。
  - CLI 命令树：`internal/cli/root/root.go:New()` 的 AddCommand 顺序为 account→config→catalog(search,detail,comments,magnets,download,tags,browse,entity×6 [actor,series,maker,director,code,list])→user(watched,want,recent,collections,mark,unmark)→rankings→top250→lists→update→version；persistent flags 仅 `--proxy`、`--host`；`Run` 先 `update.CleanupPendingWindowsUpdate()`，错误→stderr+exit 1，再 Execute 错误→stderr+exit 1。`build/javdb --help` 实测 27 顶层命令；auth 子命令 check/list/login/remove/use；rankings 子命令 actors/movies/playback；lists 子命令 related/search/show；`version --json` 输出 `{"build_date":"unknown","commit":"unknown","version":"dev"}`。
  - 公开 SDK 符号（`go doc -all sdk`）：常量 HostMirror/HostMain；函数 ActorPeriod、FilterMagnets、LoadOrCreateDeviceUUID、MagnetURI、PickBestMagnet、RankingPeriod；类型 APIError=appapi.Error、AuthRequired=appapi.AuthRequired、BrowseOptions、Client（New + API() + 全部 typed 方法）、EntityMoviesOptions、MovieMediaDownloadOptions/Result、Option（WithDeviceUUID/WithHost/WithLang/WithProxy/WithTimeout/WithToken）、SearchOptions、SearchResult。taxonomy 返回类型：`RefreshTagTaxonomy(ctx, zone) (*tags.Doc, error)`、`LoadOrRefreshTaxonomy(ctx, zone, force) (*tags.Doc, string, error)`（`*tags.Doc` 为冻结兼容例外）。错误类型：`APIError = appapi.Error`、`AuthRequired = appapi.AuthRequired`。
  - `Client.API()` 返回对象（appapi 根 Client）方法集：New、DeviceUUID、SetToken、Token，及 AllEntityMovies、Browse、Collected、CollectedPage、DeleteJSON、DownloadHLS、DownloadImage、EntityDetail、EntityMovies、GetJSON、ListInfo、LoadOrRefreshTaxonomy、Login、Mark、MovieComments、MovieDetail、MovieMagnets、MyLists、PostFormJSON、RankingsActors、RankingsMovies、RankingsPlayback、RecentViewed、RefreshTagTaxonomy、RelatedLists、ResolveEntity、ResolveMovieID、ResolveTags、ResolveUserID、ReviewMoviesPage、Search、Startup、TagsRaw、Top250、Unmark、Users、WantMovies、WatchedMovies；alias 类型 AuthRequired/BrowseOptions/EntityMoviesOptions/Error/LoginResponse/Options/SearchOptions/SearchResult。
  - 离线基线检查：go list、gopls、test、race、vet、build.sh、build 二进制冒烟、architecture/documentation 门禁、git diff --check 全部执行，只记录现状，未修复与本目标无关的问题。
- 验证证据：`go list ./...` 52 包可加载；`rg --files -g '*.go' -0 | xargs -0 gopls check`（v0.21.1）全仓无诊断；`go test ./...` 全部 ok（无 FAIL）；`go test -race ./...` 28 包 ok 且无 FAIL/panic/DATA RACE（exit 0）；`go vet ./...` exit 0；`sh scripts/build.sh` 成功产出 build/javdb（version=dev commit=3da5d6f）；`./build/javdb --help`、`auth/rankings/lists/update/version/mark --help`、`version --json` 输出如上；`sh scripts/test-architecture.sh` 与 `sh scripts/test-documentation.sh` exit 0；`git diff --check` exit 0。
- 剩余风险：
  - 当前 `scripts/test-architecture.sh` 是"迁移态"门禁：强制 facade 文件存在（cli/appapi/update/config facade、releasenotes/compat）并允许旧目录（cli/root、commands/{account,catalog,user}、input/output、shared/values）。goal-1 的 Task 11/13 必须将其切换为最终 allowlist，不能提前误删旧路径导致门禁失败。
  - 实体命令当前由 `catalog.RegisterEntityCommands` 共享 factory 生成（`list` 复用同一 factory），Task 9 需拆成六个真实命令并保留各自 Use/Short/Args/flag。
  - gopls v0.21.1 不接受 `gopls check ./...`（把 `./...` 当文件路径报错），必须用 rg 文件流 fallback；已按维护者文档记录。
  - 上一阶段迁移全部未提交（含 staged 的 docs/superpowers spec 删除）；按 input.md，goal 全程不自动 commit，工作树为唯一基线。
  - `.idea/` 为用户本地 IDE 目录，未触碰。
- 下一步：Task 2 建立公开 SDK 与 CLI 契约测试。

## Task 2：建立公开 SDK 与 CLI 契约测试

- 状态：`completed`
- 范围：先锁住本次结构迁移不得改变的公开行为。
- 必须完成：
  - 新增 SDK external-package 编译断言，覆盖全部导出类型、函数和 Client 方法签名。
  - 保留并重命名 `Client.API()` 测试，断言返回非 nil、DeviceUUID 和原方法集可调用。
  - 断言 taxonomy 返回类型、错误匹配和现有 deprecated alias。
  - 锁定根命令顺序、persistent flags、关键子命令 help、version 文本/JSON、无网络参数错误。
  - 用字面 expected strings 锁定 JSON HTML escaping、尾随换行、影片/实体/磁力/评论/列表/排行和空结果文本。
- 验证要求：`go test -race ./sdk ./internal/cli/...`、对应 vet、gopls、gofmt 和 diff check。
- 实际做了什么：
  - 新增 `sdk/contract_external_test.go`（external test package `javdb_test`）：编译期断言 HostMirror/HostMain 常量、New/WithDeviceUUID/WithHost/WithLang/WithProxy/WithTimeout/WithToken/LoadOrCreateDeviceUUID/FilterMagnets/PickBestMagnet/MagnetURI/RankingPeriod/ActorPeriod 包级函数、全部 27 个 `*javdb.Client` 方法签名，以及 SearchResult/BrowseOptions/EntityMoviesOptions/SearchOptions/Option/MovieMediaDownloadOptions/MovieMediaDownloadResult/APIError/AuthRequired 类型与 `SearchResult.Movies/Named` 方法表达式。行为测试覆盖 host 常量值、无网络 New+Token/SetToken/API()、磁力纯 helper（cnsub 过滤/最佳选择/URI）、period 映射、空 SearchResult accessor 返回 nil、API() 反射指针类型。
  - 重写 `sdk/facade_test.go`（package javdb）：`var _ func(*Client) *appapi.Client = (*Client).API` + 全量 `(*appapi.Client)` 方法值断言（含 DeviceUUID、SetToken/Token、GetJSON/PostFormJSON/DeleteJSON、全部远程方法、Startup/Users/TagsRaw）；taxonomy 冻结例外用方法值锁定 `RefreshTagTaxonomy(...) (*tags.Doc, error)` 与 `LoadOrRefreshTaxonomy(...) (*tags.Doc, string, error)`；`TestClientAPIReturnsCompatibleFacade` 断言 API() 非 nil、DeviceUUID、Token/SetToken；`TestAPIErrorAndAuthRequiredMatchErrors` 断言 errors.As 匹配 `*APIError`、`*AuthRequired` 及沿 Unwrap 匹配底层、Error() 文本；`TestDeprecatedActorPeriodAlias` 断言与 RankingPeriod 行为一致。
  - 新增 `internal/cli/contract_test.go`（package cli，经 Run/newRoot）：根 help 全量字面（锁定命令集合、cobra 显示顺序、persistent flags）；9 个关键子命令（version/update/search/detail/magnets/mark/lists/rankings/auth）`--help` 全量字面；persistent `--proxy/--host` DefValue+Usage 锁定；无网络前置参数错误精确 stderr（search/detail/mark 缺参数 `accepts 1 arg(s), received 0`、`update --json` 缺 `--check`、未知命令 `unknown command "frobnicate" for "javdb"`），断言 exit 1 且 stdout 为空。
  - 新增 `internal/cli/output/output_contract_test.go`（package output）：逐字节字面锁定 PrintMovies/PrintNamed/PrintMovieMagnets/PrintMovieComments/PrintRankedMovies/PrintNamedNoCount/PrintLists 的列分隔、尾随换行与空结果文案（`(空列表)`/`(无磁力链)`/`(无评论)`）；`EmitJSON` 锁定 SetEscapeHTML(false)、map key 字典序、恰好一个尾随换行（`{"chinese":"你好","title":"A<B>&C"}\n`）；`FmtSize` 表格锁定 2048→2.0GB、64→64MB、1024→1.0GB、`"1GB"` 降级、nil→空、0→"0"；`FilterHasMagnets` 保留缺失字段。
  - 过程中实测：cobra 的 `Command.Help()` 不渲染 `-h, --help` 行，而真实 `Run(name --help)` 会渲染，故子命令 help 字面以真实 `./build/javdb <cmd> --help` 字节为准修正；`FmtSize(float64(0))` 实际为 `"0"`（StringValue 保留原值），修正期望。
  - version 文本/JSON 已由既有 `internal/cli/version_cmd_test.go` 锁定（buildinfo 注入后校验文本与 JSON），未重复。
- 验证证据：`go test ./sdk ./internal/cli/...` 全 ok；`go test -race ./sdk ./internal/cli/...` 全 ok（无 FAIL/race）；`go vet ./sdk ./internal/cli/...` exit 0；`gopls check` 覆盖全部新增/改写测试文件无诊断；`gofmt -l` 新增文件清空（已 gofmt -w 修正 sdk 两文件与 output 契约测试的对齐）；`git diff --check` exit 0；全仓 `go test ./...` 无 FAIL/panic；`sh scripts/test-architecture.sh` 与 `sh scripts/test-documentation.sh` exit 0（SDK internal 依赖仍在 config/appapi/tags 白名单内，external 测试不导入 internal）。未运行真实凭据、真实 API、在线更新或外部写入。
- 剩余风险：
  - 新增的契约测试当前寄居在 `internal/cli`（facade 包）、`internal/cli/output` 与 `sdk` 旧结构上；Task 7–11 删除 facade/input/output/catalog 时，这些测试须随 owner 迁移（根 help/顺序契约留在最终 root 契约测试；输出字面迁往 movie/magnet/entity 与真实命令测试；SDK 契约测试保持原位），迁移后行为不得变。
  - 根 help 字面包含 cobra 自动生成的 completion/help 命令，Task 11 的最终 root.go 必须继续保留 cobra 默认 help/completion（不显式 AddCommand 即可），否则该字面会漂移并需要契约说明。
  - `update --json` 前置校验、download 空输出校验等无网络路径依赖命令 RunE 内先于 coordinator/网络执行；后续迁移命令时不得把校验后移。
- 下一步：Task 3 建立 common/jsonx 与 common/scalar。

## Task 3：建立 common/jsonx 与 common/scalar

- 状态：`completed`
- 范围：迁移 ObjectArray、ObjectSlice、RawString、MarshalLine、String、Int64，删除 `internal/shared/values`。
- 必须完成：
  - 按 plan 中固定签名建立两个子包，common 根目录不建立 package。
  - 为 nil、非法 JSON、null、非 object、数字类型、浮点截断、溢出、HTML escaping 和换行建立表格测试。
  - 逐调用方迁移 App API codec/endpoint 和 CLI 当前实现。
  - 保留 CLI StringValue 等领域差异，不把其并入 scalar。
  - 删除 shared/values 前用 gopls references 和 `rg` 确认无引用。
  - 同步架构门禁的 common 目录和依赖规则。
- 验证要求：common 聚焦 test/race/vet、所有调用包测试、全仓 `go test ./...`、gopls、架构门禁。
- 实际做了什么：
  - 新增 `internal/common/jsonx`：`ObjectArray(raw json.RawMessage) []map[string]any`、`ObjectSlice(value any) []map[string]any`（复制原 shared/values 实现，保持 nil/非法 JSON/null 元素/非 object 语义）、`RawString(raw json.RawMessage) string`（剥外层引号、不 unescape，覆盖 codec 的 map+key 包装语义之外的 CLI output.RawString 语义）、`MarshalLine(value any) ([]byte, error)`（`json.Encoder`+`SetEscapeHTML(false)`，恰好一个尾随换行，不接收 writer，编码错误原样返回）。包 doc 明确不写输出、不含 CLI 文案、不吞错。
  - 新增 `internal/common/scalar`：`String(value any) string`（fmt.Sprint，nil→""）、`Int64(value any) (int64, bool)`（含 json.Number/int/uint 全族/float32/64 截断/string，NaN/Inf/溢出返回 false）。`internal/common` 根目录无 .go 文件，不建立 Go package。
  - 表格测试：jsonx_test 覆盖 ObjectArray（nil/empty/null/[]/null 元素/非数组/非法）、ObjectSlice（nil/map slice/any slice 过滤非 map/struct slice/非 slice/chan）、RawString（quoted/chinese/number/`a\"b` 保持原始转义字节不 unescape）、MarshalLine（null/map 字典序+无 HTML 转义/array/scalar、恰好一个 `\n`、chan 返回编码错误）；scalar_test 覆盖 String 与 Int64 全表（含截断、溢出、NaN）。
  - 调用方迁移：`codec.ObjectArray/ObjectSlice` 改委托 jsonx；`output.EmitJSON` 改为 `jsonx.MarshalLine` + `w.Write`（保持 SetEscapeHTML(false) 与单尾随换行字节不变）；`output.RawString(result,key)` 改为 `jsonx.RawString(result[key])`（语义一致，剥引号不 unescape）；`output.StringValue` 保留 float64→int 截断领域差异，仅 default 分支改用 `scalar.String`；`endpoint/{browse,entity,magnets,movie/resolve,user}` 的 `values.String` → `scalar.String`。
  - 删除 `internal/shared/values/`（含其包内测试）：先 `rg` 确认生产代码与测试零引用，`go list ./...` 不再出现该包。
  - 同步架构门禁：required_dirs 以 `internal/common/{jsonx,scalar}` 替换 `internal/shared/values`；新增 `internal/common` 根目录不得含 Go 文件、common 不得 import CLI/SDK/App API/config/update 两条规则。
- 验证证据：`go test ./internal/common/... ./internal/javdb/appapi/... ./internal/cli/... ./sdk/` 全 ok；`go test -race ./internal/common/... ./internal/javdb/appapi/codec ./internal/cli/output` ok；`go vet` 涉及包 exit 0；`gopls check` 覆盖全部新增/迁移文件无诊断；`gofmt -l` 对 common/endpoint/output 清空（browse/entity/movie/resolve/user 五文件 perl 替换后 gofmt -w 修正 import 分组）；`git diff --check` exit 0；全仓 `go test ./...` 无 FAIL/panic；`go doc` 两新包导出面符合 plan；`sh scripts/test-architecture.sh` 与 `sh scripts/test-documentation.sh` exit 0。未运行真实凭据、真实 API、在线更新或外部写入。
- 剩余风险：
  - `codec.RawString(m,key)`（map+key 签名，json.Unmarshal 转义语义）未迁入 jsonx，仍留在 codec 包：它与 jsonx.RawString 的"不 unescape"语义不同，为保持 auth/user 提取行为未合并；Task 4 的 appapi 重构需保持这一点，否则用户名含转义时行为漂移。
  - `internal/shared/values` 已删除，但维护者文档（architecture.md/development.md）仍在描述 shared values；Task 13 统一更新文档时需改为 common/jsonx+scalar 描述。
  - `output.EmitJSON` 现经 `jsonx.MarshalLine` 中转（多一次 buffer copy），输出字节与错误传播语义不变；性能非目标，Task 8+ 迁移真实命令直接调用 jsonx 时可再评估。
- 下一步：集中检查-debug 1。

## 集中检查-debug 1：基线、契约与 common

- 状态：`completed`
- 检查范围：需求偏离、公开契约覆盖、common 边界、重复/死代码、LSP、包图、测试、安全和工作树保留。
- 必须完成：发现问题时在后续 task 前追加精确修复 task，不把已知问题留给 App API 迁移掩盖。
- 实际做了什么：以 Task 1–3 当前工作树执行集中复查。需求偏离：Task 3 仅新建 common 两子包、迁移调用方、删除 shared/values、更新门禁，未改公开 CLI/SDK 行为、未动用户文件。公开契约：Task 2 的 SDK external/API() 契约与 CLI 根 help/子命令 help/无网络参数错误/输出字面契约全部保持通过；`output.EmitJSON` 改为经 `jsonx.MarshalLine` 中转后字节与错误语义不变（契约测试证明）。common 边界：`go list -f` 确认 jsonx 仅依赖 bytes/encoding/json、scalar 仅依赖 encoding/json/fmt/math/strconv；`internal/common` 根目录 0 个 .go 文件；门禁新增 common 无上层依赖与 common 根无包两条规则。重复/死代码：`internal/shared/values` 已删除且全仓（含 test）零引用，`rg` 无 `shared/values`/`shared.` 残留；jsonx/scalar 成为唯一实现源；CLI `StringValue` 的 float64→int 截断领域差异保留在 output，未并入 scalar；`codec.RawString` 的 unescape 语义按计划保留在 codec（与 jsonx.RawString 不 unescape 区分）。LSP：`rg --files -g '*.go' -0 | xargs -0 gopls check` 全仓无诊断。包图：`go list ./...` 53 包可加载无循环（Task 1 实际基线为 52 包，含 shared/values；本轮 -1 shared/values +2 common = 53；已更正 Task 1 记录中的包数笔误 47→52）。测试：全仓 `go test ./...`、`go vet ./...` 通过；common/调用包 `go test -race` 通过；`gofmt -l` 全仓清空；`git diff --check` exit 0；`sh scripts/test-architecture.sh`、`sh scripts/test-documentation.sh` exit 0。安全：common 为纯函数，无网络、无 secret、无 IO 输出。工作树：用户既有文件（README、双语 CLI/SDK 文档、changelog、sdk/rankings.go、skill、.idea/）保持不动。未发现需要追加修复 task 的代码问题。
- 验证证据：全仓 gopls 无诊断；`go test ./...` 无 FAIL；`go test -race ./internal/common/... ./internal/javdb/appapi/codec ./internal/cli/output` 通过；`go vet ./...`、`gofmt -l $(rg --files -g '*.go')`、`git diff --check` 均 clean；`sh scripts/test-architecture.sh` 与 `sh scripts/test-documentation.sh` exit 0；`go list ./...` 53 包无循环；common 依赖检查仅 stdlib。
- 剩余风险：
  - `codec.RawString`（unescape 语义）与 `jsonx.RawString`（不 unescape）并存是刻意保留的差异；Task 4 重构 App API 时不得无意统一。
  - 维护者文档仍在描述 shared values，Task 13 统一改为 common 描述。
  - 无其他已知问题需要追加修复 task。
- 下一步：Task 4。

## Task 4：将 App API endpoint 改为 capability service

- 状态：`completed`
- 范围：建立 endpoint service、真实根 Client 组合，迁移 SDK 内部依赖和 login_probe，删除 facade/forwarder。
- 必须完成：
  - 建立 Auth/Browse/Entity/Lists/Movie/Rankings/Search/User/Media Endpoint 类型与构造器。
  - 按固定构造顺序显式注入 transport 和 endpoint 间依赖。
  - 根 Client 通过未导出别名嵌入 capability，使用 method promotion，避免暴露字段和手写转发。
  - magnets 保持纯 helper。
  - SDK 直接引用 model 和必要纯 helper，对外签名不变。
  - login_probe 改用 SDK + API()，只编译不真实运行。
  - 删除 appapi facade 与 facade test，新增 Client 构造、method-set、错误和关键 helper 测试。
  - 同步架构门禁的 App API/SDK import allowlist。
- 验证要求：App API/SDK/login_probe 聚焦 test/race/vet、gopls、`go doc`、全仓测试和架构门禁。
- 实际做了什么：
  - 将 `endpoint/{auth,browse,search,lists,rankings,movie,entity,user}` 从包级函数（`func X(c *client.Client, ...)`）重构为有状态 capability service：`AuthEndpoint`、`BrowseEndpoint`、`SearchEndpoint`、`ListsEndpoint`、`RankingsEndpoint`、`MovieEndpoint`、`EntityEndpoint`、`UserEndpoint`，各自构造器持 transport 指针；`media.MediaEndpoint` 持 `Fetch` 回调。`endpoint/magnets` 保持纯 helper，未制造 service。
  - 依赖注入按 plan 固定顺序：`transport → auth/browse/search/lists/rankings → movie(search) → entity(browse, search) → user(movie) → media(fetcher)`。构造顺序在 `appapi.New` 中显式排列：`NewAuth(t)`、`NewBrowse(t)`、`NewSearch(t)`、`NewLists(t)`、`NewRankings(t)`、`NewMovie(t, s)`、`NewEntity(t, b, s)`、`NewUser(t, m)`、`NewMedia(t.FetchMedia)`。
  - 根 `appapi.Client` 重写为真实组合层（`client.go`）：定义 10 个未导出**指针**类型别名（`transportClient = *client.Client`、`authClient = *auth.AuthEndpoint` 等），嵌入到 `Client` struct；method promotion 提供全部扁平方法，字段名不可访问（未导出别名），无手写 forwarder。`New` 只构造一次 transport。每轮 `sdk.New` 只经 `appapi.New` 建一个 transport。
  - 包级纯 helper 保留在根包：`AllPages`、`BuildSearchParams`、`BuildTagFilter`、`BuildEntityFilter`、`SearchTypeListKey`、`ResolveNumber`、`RankingPeriod`、`ActorPeriod`、`BuildTop250Params`、`FilterMagnets`、`PickBestMagnet`、`MagnetURI`、`LoadOrCreateDeviceUUID`；`Zones` 直接 `model.Zones`，`MainFlags/EntityLetters` 指向 `browse.MainFlags/browse.EntityLetters`（同一 map 身份），`CollectionSpecs` 指向 `user.CollectionSpecs`。
  - SDK 内部调用不变：`sdk/*.go` 仍导入根 `appapi`，`New` 仍经 `appapi.New`；对外类型名与签名不变（`go test ./sdk` 契约测试通过）。SDK 不新增 internal 类型泄漏。
  - `scripts/login_probe.go` 改为 `javdb "github.com/FlanChanXwO/javdb-cli/sdk"` + `javdb.New(javdb.WithHost(javdb.HostMirror))` + `client.API()` 后调用 `Login/Users/Startup/ResolveUserID`；仅编译不运行。
  - 删除 `internal/javdb/appapi/facade.go` 与 `facade_test.go`；新增 `internal/javdb/appapi/client_test.go`：`New` 构造断言（无网络、DeviceUUID/Token/SetToken 状态）、method-set 方法值断言（全部 42 个方法）、`TestCapabilitiesShareTransportState`（根 Client SetToken 后经 auth capability 的 `ResolveUserID` 本地 JWT 解析返回 user_id=123，证明同一 transport 指针）、`TestHelperReexportsStable`（SearchTypeListKey/RankingPeriod/ActorPeriod/MagnetURI/Zones/MainFlags/EntityLetters/CollectionSpecs/FilterMagnets/ResolveNumber/BuildSearchParams/BuildTop250Params/AllPages）、`TestErrorAndAuthRequiredMatch`、`TestSearchResultAccessorsStable`。`TestNewRejectsInvalidProxy` 对 tls-client 构造错误做 `t.Skip` 降级（避免依赖本地 tls-client 版本行为）。
  - 同步架构门禁：facade_file 列表移除 `internal/javdb/appapi/facade.go`，新增要求 `client.go` 存在且 `facade.go` 不存在；原有"endpoint 不反向导入根 appapi/SDK/CLI"与"SDK 只导入 config|appapi 根|storage/tags"规则继续生效。
- 验证证据：`go test ./...` 无 FAIL/panic；`go test -race ./internal/javdb/appapi/... ./sdk/` 无 FAIL/DATA RACE；`go vet ./...` exit 0；`rg --files -g '*.go' -0 | xargs -0 gopls check` 全仓无诊断；`gofmt -l internal/javdb/appapi scripts/login_probe.go` 清空；`go test ./scripts/login_probe.go` 编译通过（`? command-line-arguments [no test files]`）；`go doc` 根 appapi 显示真实 Client 组合层 doc；`go doc -all sdk` 无 internal 类型泄漏；`sh scripts/test-architecture.sh`、`sh scripts/test-documentation.sh`、`git diff --check` 均 exit 0；`rg` 确认 endpoint 包不导入根 appapi/SDK/CLI。未运行真实凭据、真实 API、在线更新或外部写入。
- 剩余风险：
  - `go doc -all` 对未导出嵌入字段的 promoted 方法不渲染（Go doc 行为）；`Client.API()` 方法集的权威证明是 `sdk/facade_test.go` 的方法值编译断言（全量 42 方法）与 `internal/javdb/appapi/client_test.go` 方法值断言，均已通过。
  - `media.DownloadImage/DownloadHLS` 保留包级纯函数（`func DownloadImage(fetch Fetch, ...)`）+ `MediaEndpoint` 薄方法，避免重复实现；这是 Task 4 范围内唯一的方法包装（media 不是 endpoint 子包，plan 亦如此）。
  - endpoint capability 构造器现在导出（`NewAuth` 等），供根包与将来测试使用；根 Client 通过未导出别名嵌入，字段不可访问，符合 plan。
  - 维护者文档（architecture.md）仍在描述"appapi 根 facade"，Task 13 统一更新为真实 Client 组合层描述。
- 下一步：Task 5。

## Task 5：删除 config 根 facade

- 状态：`completed`
- 范围：让所有调用方直接依赖 paths/settings，保持配置与路径行为。
- 必须完成：
  - SDK 的 host 常量和 URL 映射改用 settings。
  - CLI app/config 命令改用 paths/settings。
  - tags storage 改用 paths。
  - 根 settings 测试迁至 settings；删除 facade 与 facade test。
  - 用 gopls references 确认根包无调用方后使 `internal/config` 根目录不含 Go 文件。
  - 同步架构门禁。
- 验证要求：config、SDK、storage、CLI 聚焦 test/race/vet、gopls、全仓测试和架构门禁。
- 实际做了什么：
  - `sdk/client.go` 改导入 `internal/config/settings`，`HostMirror/HostMain` 仍为 settings 常量（`"mirror"/"main"`，对外契约由 external contract 测试锁定），`New` 的 host→URL 映射改 `settings.HostURLs`。
  - `internal/cli/app/app.go` 改 `paths.ConfigPath/DeviceUUIDPath/AuthPath/EnsureDir` + `settings.LoadFile/ValidateHost/Resolve`，`Runtime` 类型改 `settings.Runtime`。
  - `internal/cli/app/update.go` 改 paths/settings，本地变量 `settings`→`cfg` 避免遮蔽包名。
  - `internal/cli/commands/config/config.go` 的 path/get/set/unset 全部改用 `paths.ConfigPath`、`settings.LoadFile/ValidateHost/SaveFile/HostMirror`，本地 `settings`→`cfg`。
  - `internal/storage/tags/file.go` 改 `paths.TagTaxonomyPath`。
  - `internal/cli/facade.go`（仍为兼容 facade，Task 7 删除）改 `config.Runtime`→`settings.Runtime`。
  - `internal/config/settings_test.go` 从 `package config` 迁至 `internal/config/settings/settings_test.go`（`package settings`），保留 TestResolvePrecedence/TestSaveLoad，并新增 TestValidateHost（覆盖空/mirror/URL/bogus）。
  - 删除 `internal/config/facade.go`、`facade_test.go`、旧 `settings_test.go`；`internal/config` 根目录现在只有 paths/settings 子目录、0 个 .go 文件，`go list` 不再出现 `internal/config` 包。
  - 同步架构门禁：facade_file 列表移除 `internal/config/facade.go`，新增 `internal/config` 根目录不得含 .go 文件规则。
- 验证证据：`go build ./...` exit 0；`go test ./...` 无 FAIL/panic；`go test ./internal/config/... ./internal/storage/tags/... ./internal/cli/... ./sdk/` 全 ok；`go test -race ./internal/config/... ./sdk/` 无 FAIL/DATA RACE；`go vet ./...` exit 0；`rg --files -g '*.go' -0 | xargs -0 gopls check` 全仓无诊断；`gofmt -l internal/config internal/cli sdk internal/storage/tags` 清空；`git diff --check` exit 0；`sh scripts/test-architecture.sh`（新增 config 根无 Go 文件规则通过）与 `sh scripts/test-documentation.sh` exit 0；`rg` 确认全仓无 `internal/config"` 导入；`build/javdb config --help` 与 root help 冒烟通过；SDK Host 常量值由 contract 测试验证仍为 mirror/main。未运行真实凭据、真实 API、在线更新或外部写入。
- 剩余风险：
  - `settings.SaveFile` 内部调用 `paths.EnsureDir()` 会创建真实 `~/.javdb-cli` 目录（无论 path 参数），这是既有行为，测试沿用未改；Task 13 文档更新时可在 maintenance notes 注明。
  - 维护者文档仍在描述 `internal/config` 根 facade，Task 13 统一更新为 paths/settings。
- 下一步：Task 6。

## Task 6：删除 update facade 与根 alias types

- 状态：`completed`
- 范围：保留真实 Coordinator，以其自有接口组合 update 子包。
- 必须完成：
  - 在 interfaces.go 定义 Coordinator 所需最小接口，不 alias 子包类型。
  - Execute 直接使用 model.Request/Result。
  - CLI app 直接构造 release/source/process/archive 生产依赖。
  - CLI root 直接调用 process Windows cleanup；命令与状态输出使用 model。
  - 删除 facade、facade test、根 types.go 和旧调用路径。
  - 保持错误文本、安装来源、SemVer、checksum 和替换行为。
  - 同步架构门禁。
- 验证要求：update/CLI 聚焦 test/race/vet、Windows 文件 gopls、全仓测试和架构门禁。
- 实际做了什么：
  - 新增 `internal/update/interfaces.go`：Coordinator 自有最小接口 `SourceDetector`（`Detect(buildinfo.Info) (model.InstallSource, error)`）、`SourceDetectorFunc`、`ReleaseChecker`（`Check(context.Context, bool) (*model.Release, error)`）、`CommandRunner`、`ReleaseInstaller`（`Install(context.Context, model.Release) error`），全部直接使用 `model.*` 类型，不 alias 子包接口。
  - `coordinator.go`：`CoordinatorOptions` 用根自有接口；`Execute(ctx, model.Request) (model.Result, error)`；`InstallSource*` 常量改 `model.InstallSource*`；`model.HomebrewFormula`/`model.GoInstallPackage` 直接用。错误文本逐字保持（development 拒绝、Homebrew/go-install/release 分支、unsupported source）。
  - `coordinator_test.go`：改用 `model.Request/Result/Release/InstallSource*`；fake checker/installer 同步 model 类型；四个测试行为不变。
  - `internal/cli/app/update.go`：`NewProductionUpdateCoordinator` 直接构造 `release.NewReleaseHTTPClient(proxy)`、`release.NewGitHubReleaseClient(release.ReleaseClientOptions{HTTPClient})`、`update.NewCoordinator(update.CoordinatorOptions{SourceDetector: update.SourceDetectorFunc(source.DetectInstallSource), ReleaseChecker: releaseClient, CommandRunner: process.NewCommandRunner(stdout, stderr), ReleaseInstaller: archive.NewReleaseInstaller(archive.ReleaseInstallerOptions{HTTPClient})})`。
  - `internal/cli/root/root.go`：`process.CleanupPendingWindowsUpdate()`。
  - `internal/cli/output/update.go`：`model.Result`。
  - `internal/cli/commands/update/update.go`：`model.Request`。
  - 删除 `internal/update/facade.go`、`facade_test.go`、`types.go`（根 alias types）。根包现在只有 coordinator.go/interfaces.go/coordinator_test.go。
  - 同步架构门禁：facade_file 列表移除 `internal/update/facade.go`，新增要求 coordinator.go/interfaces.go 存在、facade.go 与 types.go 不存在。
- 验证证据：`go build ./...` exit 0；`go test ./...` 无 FAIL/panic；`go test ./internal/update/... ./internal/cli/...` 全 ok；`go test -race ./internal/update/... ./internal/cli/...` 无 FAIL/DATA RACE；`go vet ./...` exit 0；`rg --files -g '*.go' -0 | xargs -0 gopls check` 全仓无诊断（含 Windows 文件）；`gofmt -l internal/update internal/cli` 清空；`git diff --check` exit 0；`sh scripts/test-architecture.sh` 与 `sh scripts/test-documentation.sh` exit 0；`build/javdb update --help` 与 `update --json`（无 --check → exit 1 + `--json is only supported with --check`）冒烟通过；根 update 契约测试（root/root_test.go、internal/cli contract）通过。未运行真实凭据、真实 API、在线更新（`update --check` 未执行）或外部写入。
- 剩余风险：
  - `internal/cli/facade.go` 仍引用 `app.LoadRuntime` 等兼容 shim，Task 7/11 删除该 facade 时需同步移除 `rootFlags/appIO/toAppFlags/toAppIO` 等私有 shim。
  - 维护者文档仍在描述 `internal/update` 根 facade 与 alias，Task 13 统一更新为 Coordinator+interfaces 描述。
  - `source`/`release`/`archive` 子包内部仍有自己的 `type Release = model.Release` 等便利 alias（包内使用，非根兼容入口），plan 允许各领域包按需引用 model。
- 下一步：集中检查-debug 2。

## 集中检查-debug 2：App API、SDK、config 与 update

- 状态：`completed`
- 检查范围：公开 SDK、API() 方法集、endpoint 依赖、配置行为、更新行为、平台文件、包图、LSP、测试、安全与文档缺口。
- 必须完成：确认没有为删除 facade 而复制实现或扩大 SDK internal 泄漏。
- 实际做了什么：
  - 公开 SDK：`go doc -all sdk` 无 internal 类型泄漏（门禁 go doc 检查通过）；external contract 测试与 facade_test 全绿。
  - API() 方法集：`sdk/facade_test.go` 的 `(*appapi.Client)` 全量方法值断言编译通过（42 方法），根 Client method promotion 未丢方法。
  - endpoint 依赖：`rg` 确认 endpoint/{auth,browse,entity,lists,movie,rankings,search,user} 不导入根 appapi、SDK 或 CLI；构造顺序与 plan 一致（transport→auth/browse/search/lists/rankings→movie(search)→entity(browse,search)→user(movie)→media(fetcher)）。
  - 配置行为：`settings` 测试（flag>env>file>default 优先级、save/load、ValidateHost）全绿；config 根已无 Go 文件、无导入者。
  - 更新行为：Coordinator 四个测试（release only when newer、check 不安装、go-install/brew 分支、development 拒绝）全绿；错误文本保持。
  - 平台文件：`replace_windows.go`/`replace_nonwindows.go` 等 Windows/平台文件纳入全仓 gopls 检查（无诊断）。
  - 包图：`go list ./...` 52 包无循环；`internal/config` 根包已消失。
  - 无复制实现：root appapi 的 `AllPages/BuildSearchParams/RankingPeriod/FilterMagnets/MagnetURI` 等均为单行 delegate 到 endpoint 源实现，无重复逻辑；`internal/update` 根包只剩 coordinator/interfaces/测试。
  - SDK internal 泄漏未扩大：SDK 仅导入 `internal/config/settings`、根 `internal/javdb/appapi`、`internal/storage/tags`，仍在门禁 allowlist 内，未新增 client/model/codec/media/endpoint 深层依赖。
  - LSP/测试/安全：全仓 gopls 无诊断；`go test ./...`、`go test -race ./...`、`go vet ./...`、build 全部通过；无新增 secret 输出（account.go 的 password 是既有登录流程）；未运行真实 API/凭据/在线更新/apply。
  - 文档缺口：维护者 architecture.md/development.md 仍描述旧的"appapi 根 facade / config 根包 / update 根 facade"结构，Task 13 统一更新。
- 验证证据：全仓 gopls 无诊断；`go test ./...` 无 FAIL；`go test -race ./...` exit 0（无 FAIL/DATA RACE/panic）；`go vet ./...`、`gofmt -l`、`git diff --check` 全 clean；`scripts/test-{architecture,documentation,releasenotes,package-release,homebrew-formula,workflows}.sh` 全部 exit 0；`go list ./...` 52 包；`build/javdb --help`、`version --json`、`config path` 冒烟通过。
- 剩余风险：无需要追加修复 task 的代码问题；仅保留 Task 13 的维护者文档更新义务。
- 下一步：Task 7。

## Task 7：建立 CLI app、movie、magnet、entity 与基础命令

- 状态：`completed`
- 范围：建立最终共享边界，迁移 auth/config/update/version 和 prompt。
- 必须完成：
  - app 只管理 IO、root flags、runtime、SDK client、auth 和 update dependency assembly。
  - movie 按固定 API 提供影片 Row、Project、ProjectAll、FilterHasMagnets。
  - magnet 按固定 API 提供 Magnet Row、Project、ProjectAll、FormatSize。
  - entity 按固定 API提供 Options、Result、Execute、NamedRow 投影。
  - auth 目录拥有真实 auth command 和 prompt；config/update/version 使用同名主文件。
  - 不删除旧 root/catalog/user/output，保持过渡期间全仓可编译。
  - 为所有新包补聚焦测试。
- 验证要求：新 CLI 包 test/race/vet、gopls、全仓测试；当前架构门禁按过渡状态同步但不得放宽最终禁止项。
- 实际做了什么：
  - 确认 `internal/cli/app` 已只负责 IO/root flags/runtime/SDK client/auth/update dependency assembly（Task 5/6 迁移后无业务逻辑残留）。
  - 新增 `internal/cli/movie`：`Row{Number,ID,Title,ReleaseDate}`、`Project`、`ProjectAll`、`FilterHasMagnets`；`display` 保留 CLI 数值 ID 截断约定（float64→int），`intValue` 用 scalar.Int64 兜底。包无 IO/Cobra/SDK/JSON。
  - 新增 `internal/cli/magnet`：`Row{Name,Size,Flags,CreatedAt,Hash}`、`Project`、`ProjectAll`、`Flags`、`FormatSize`；`FormatSize` 与 output.FmtSize 字面一致（2048→2.0GB、64→64MB、1024→1.0GB、"1GB" 降级、nil→空、0→"0"）。包只返回结构化 Row。
  - 新增 `internal/cli/entity`：`Options`、`Result`、`Execute(ctx, *javdb.Client, kind, ref, Options)`（统一 ResolveEntity/ResolveTags/EntityMovies 或 AllEntityMovies(50)/FilterHasMagnets/EntityDetail+`{"id":eid}` 降级）、`NamedRow`、`ProjectNamed`、`ProjectNamedAll`。包不创建 Cobra、不写输出。
  - 新增 `internal/cli/commands/auth/{auth,login,list,use,remove,check,prompt}.go`：真实 auth 命令组（login/list/use/remove/check）与 prompt 实现；`commands/account/account.go` 改为 3 行 delegate 到 `authcmd.New`（过渡 seam，Task 11 删除）。
  - config/update/version 命令目录确认已使用同名主文件（config.go/update.go/version.go）。
  - 测试：movie（Project/ProjectAll/float ID 截断/FilterHasMagnets 数值变体含 json.Number）、magnet（Project name/title 降级、日期截断、FormatSize 表格、Flags）、entity（ProjectNamed 中文优先/count 回退/无 count、ProjectNamedAll + httptest 覆盖 Execute 解析→查询→metadata 主路径与 metadata 降级 `{"id":eid}`）、auth（命令树 5 子命令、login flag 集）。
  - 未删除旧 root/catalog/user/output/input，过渡期全仓可编译。
  - 同步架构门禁：required_dirs 增加 `internal/cli/{movie,magnet,entity}` 与 `internal/cli/commands/auth`（过渡态，不放松最终禁止项）。
- 验证证据：`go build ./...` exit 0；`go test ./...` 无 FAIL/panic；`go test ./internal/cli/movie/ ./internal/cli/magnet/ ./internal/cli/entity/ ./internal/cli/commands/auth/` 全 ok；`go test -race` 相关包无 FAIL/DATA RACE；`go vet ./...` exit 0；`rg --files -g '*.go' -0 | xargs -0 gopls check` 全仓无诊断；`gofmt -l internal/cli` 清空；`git diff --check` exit 0；`sh scripts/test-architecture.sh` 与 `sh scripts/test-documentation.sh` exit 0；`build/javdb auth --help` 冒烟（5 子命令）与 root help 契约测试通过。未运行真实凭据、真实 API、在线更新或外部写入。
- 剩余风险：
  - `commands/auth/prompt.go` 与 `internal/cli/input/prompt.go` 在过渡期并存（input 仍被 facade.go 的 PromptUsername/PromptPassword shim 引用）；Task 11 删除 input 与 facade 后 auth/prompt 成为唯一源，期间两实现逐字一致。
  - `commands/account` 现在是 auth 的薄 delegate，Task 11 删除目录并切换 root 注册。
  - `movie/magnet` 的 `display`/`intValue` 与 output 包内部 helper 暂时并存（行为一致）；Task 11 删除 output 后由 movie/magnet 成为唯一源。
  - entity.Execute 依赖具体 `*javdb.Client`，离线测试通过 httptest 覆盖主路径与降级；Task 9 继续补单页/全页/错误路径。
- 下一步：Task 8。

## Task 8：迁移影片目录命令

- 状态：`completed`
- 范围：迁移 search、detail、comments、magnets、download、tags、browse。
- 必须完成：
  - 每个目录拥有同名主文件、真实 Cobra 构造器和 owner 测试。
  - search/browse 使用 movie/entity 投影；detail/magnets 使用 magnet 投影。
  - comments 保留完整内容和字段降级，不新增截断。
  - download 保留本地参数校验先于网络访问及媒体选择语义。
  - JSON 全部使用 jsonx.MarshalLine，命令传播编码和 writer 错误。
  - 不修改旧 root 注册，先让新包独立通过测试。
- 验证要求：七个命令包及共享包 test/race/vet、gopls、help/输出契约、全仓测试。
- 实际做了什么：
  - 新增 `internal/cli/commands/{search,detail,comments,magnets,download,tags,browse}` 七个真实命令包，各自同名主文件与 Cobra 构造器 `New(flags, aio)`；未修改旧 root 注册，旧 catalog 命令继续生效。
  - search：movies 分支用 `movie.ProjectAll`+`Row.Line()`、`movie.FilterHasMagnets`；命名分支用 `entity.ProjectNamedAll`+`NamedRow.Line()`；本地 `searchTypeKey` 保留原 `output.SearchTypeKey` 映射。JSON 用 `jsonx.MarshalLine`+写入并传播错误。
  - browse：`movie.FilterHasMagnets`+`movie.ProjectAll` 投影；JSON 同 search。
  - detail：`detail.go` 命令 + `detail_lines.go`（`renderDetail` 保留 番号/id/标题/评分/日期/磁力数/系列/厂牌/导演/演员/标签 全部 graph 行；`renderMagnets` 用 magnet.Row.Line/HashLine）。`display` 保持数值 ID 截断约定。
  - comments：`writeComments` 保留完整内容与 user_name/username/user_nickname/user.name 等字段降级，无截断；`--page/--limit` 正值校验在网络前。
  - magnets：`ParseSizeMiB` 迁入本包；`magnetCount` 用 scalar.Int64 兜底；best/JSON/文本分支保持；空结果 `(无磁力链)`。
  - download：本地媒体路径校验（至少一个、非空白）在网络前；DownloadMovieMedia 语义不变。
  - tags：taxonomy 缓存命令与 `taxonomy 已写入`/`(空列表)` 文案不变。
  - 在 movie/magnet/entity 投影包上增加纯 `Line()` 方法（`movie.Row.Line`、`magnet.Row.Line`/`HashLine`、`entity.NamedRow.Line`），返回不含尾随换行的行文本；空列表文案仍由命令持有（符合"包不含空列表文案"），从而避免在 8+ 命令间重复同一行格式（input.md"重复代码最少"）。
  - owner 测试：search（help flags、缺参错误、type key）、detail（help、缺参、renderDetail 全行、renderMagnets 空）、comments（help、--page 0 前置校验、writeComments 字面）、magnets（help、ParseSizeMiB、writeMagnets 空）、download（help、无媒体输出前置错误、空白输出拒绝）、tags（help flags）、browse（help flags、writeMovieRows 空）。
  - 同步架构门禁：required_dirs 增加 `internal/cli/commands/{search,detail,comments,magnets,download,tags,browse}`。
- 验证证据：`go build ./...` exit 0；`go test ./...` 无 FAIL/panic；`go test ./internal/cli/commands/... ./internal/cli/movie/ ./internal/cli/magnet/ ./internal/cli/entity/` 全 ok；`go test -race` 相关包无 FAIL/DATA RACE；`go vet ./...` exit 0；`rg --files -g '*.go' -0 | xargs -0 gopls check` 全仓无诊断；`gofmt -l internal/cli` 清空；`git diff --check` exit 0；`sh scripts/test-architecture.sh` 与 `sh scripts/test-documentation.sh` exit 0。未运行真实凭据、真实 API、在线更新或外部写入。
- 剩余风险：
  - 新命令包尚未接入 root（Task 11 切换）；期间旧 catalog 命令继续提供行为，新包以 owner 测试独立锁定帮助/flag/输出格式。
  - `writeMovieRows`/`writeJSON` 在 search 与 browse 内各自存在（约 12 行/包）；行格式本身已由 movie.Row.Line 单一来源，命令内只保留写循环与空列表文案，属于 plan"每个命令持有文本"的边界内重复。
  - detail/magnets 各自的 magnet 文本写循环同理（行格式由 magnet.Row.Line 单一来源）。
- 下一步：Task 9。

## Task 9：迁移六个实体命令

- 状态：`completed`
- 范围：迁移 actor、series、maker、director、code、list。
- 必须完成：
  - 每个命令包独立拥有 Use/Short/Args、flags、New 和同名主文件。
  - 共享逻辑只调用 entity.Execute 与 movie 投影，不建立通用 command factory 或 wrapper。
  - 保持 kind、help、flag 默认值、all-pages、tag、main、sort/order 和 JSON payload。
  - 为每个命令添加至少一个构造/help 测试；entity 用例覆盖单页、全页、metadata 降级和错误路径。
- 验证要求：实体命令与 entity/movie test/race/vet、gopls、全仓测试。
- 实际做了什么：
  - 新增 `internal/cli/commands/{actor,series,maker,director,code,list}` 六个真实命令包，各自独立 `New(flags, aio)`、Use/Short/Args（`actor REF`/`series REF`/`maker REF`/`director REF`/`code REF`/`list REF`，Short 逐字同 root help 契约）、同名主文件；无通用 command factory/wrapper。
  - 每个 RunE：`app.LoadRuntime`+`app.NewClient` → `entity.Execute(ctx, c, kind, ref, entity.Options{...})` → JSON（`{"entity","entity_id","movies"}` 经 jsonx.MarshalLine）或影片文本（`movie.ProjectAll`+`Row.Line`，空→`(空列表)`）。
  - flag 默认值保持 probe 锁定：zone=censored、tag/main 可重复、sort=release、order=desc、page=1、limit=20、all/has-magnets/json=false。
  - 每包 owner 测试：构造断言（name/Use/全 flag 存在、sort/limit 默认值）+ 缺参错误（`accepts 1 arg(s), received 0`）。
  - entity_test 补齐用例：`TestExecuteAllPages`（页 1 有数据、页 2 空停止，maxPage=2）、`TestExecuteHasMagnetsFilter`（丢弃 magnets_count==0）、`TestExecutePropagatesResolveError`（详情+搜索均 NotFound → 错误上抛）；与 Task 7 已有的单页主路径、metadata 降级 `{"id":eid}` 共同覆盖 plan 要求的单页/全页/降级/错误路径。
  - 同步架构门禁：required_dirs 增加 `internal/cli/commands/{actor,series,maker,director,code,list}`。
- 验证证据：`go test ./...` 无 FAIL/panic；`go test ./internal/cli/entity/ ./internal/cli/commands/...` 全 ok；`go test -race ./internal/cli/entity/ ./internal/cli/commands/...` 无 FAIL/DATA RACE；`go vet ./...` exit 0；`rg --files -g '*.go' -0 | xargs -0 gopls check` 全仓无诊断；`gofmt -l internal/cli` 清空；`git diff --check` exit 0；`sh scripts/test-architecture.sh` 与 `sh scripts/test-documentation.sh` exit 0。未运行真实凭据、真实 API、在线更新或外部写入。
- 剩余风险：
  - 六个实体命令 RunE 结构相似（kind/Use/Short 不同，逻辑一致调用 entity.Execute）；plan 明确"不建立通用 command factory"，此为预期设计。
  - 每个命令包内各有本地 `writeMovieRows`/`writeJSON`（行格式由 movie.Row.Line 单一来源），与 Task 8 的 search/browse 一致，属 plan"每个命令持有文本/JSON"边界内的写循环。
- 下一步：集中检查-debug 3。

## 集中检查-debug 3：CLI 共享边界与第一批命令

- 状态：`completed`
- 检查范围：命令真实性、文件命名、help/flags、远程依赖、movie/magnet/entity 职责、重复代码、文本/JSON、LSP、测试和 secret。
- 必须完成：确认 movie 没有吸收磁力/排行/IO，magnet 没有成为输出包，entity 没有成为 Cobra facade。
- 实际做了什么：
  - 命令真实性：`internal/cli/commands/{auth,search,detail,comments,magnets,download,tags,browse,actor,series,maker,director,code,list,config,update,version}` 全部真实命令/命令组，17 个目录均有与目录同名主文件；`rg --files -g 'command.go' -g 'output.go' -g 'printer.go' -g 'render.go'` 在 commands 下零命中（无泛化主文件）。
  - 共享边界（`go list -f` 依赖检查）：movie 仅 scalar+strconv（无 io/cobra/sdk/JSON/排行/磁力/详情/空列表文案）；magnet 仅 fmt+scalar+strconv（无 io/cobra/sdk/JSON writer，未成为输出包）；entity 仅 movie+scalar+sdk（Execute 需 SDK，无 cobra/io/output，未成为 Cobra facade）。
  - 重复代码：movie/magnet/entity 的 `Line()` 纯方法（`movie.Row.Line`、`magnet.Row.Line/HashLine`、`entity.NamedRow.Line`）是行格式单一来源；命令内仅保留写循环与空列表文案（plan"每个命令持有文本/JSON"边界内）。search/browse 与六个实体命令内各有本地 `writeMovieRows`/`writeJSON` 小写循环，属预期设计。
  - 远程依赖：`rg` 确认 `internal/cli/commands|movie|magnet|entity` 无 `internal/javdb/(appapi|protocol)` 导入；远程操作全部经公开 sdk。
  - 文本/JSON：jsonx.MarshalLine 统一；`(空列表)`/`(无磁力链)`/`(无评论)` 文案在命令 owner；EmitJSON 契约测试（output_contract_test.go）仍通过。
  - help/flags：root help 契约测试（internal/cli contract_test.go）通过，新命令 Use/Short/flags 与 probe 锁定一致；实体 flag 默认值（sort=release/limit=20）由 actor_test 断言。
  - LSP/测试/secret：全仓 gopls 无诊断；`go test ./...`、`go test -race ./...`、`go vet ./...`、build 全绿；`rg` 确认新命令中只有 auth 命令处理凭据（既有逻辑），无新增 secret 输出。
  - 工作树：README/sdk/rankings.go/skill/.idea/ 等用户文件未动；goal-1 三文件 + archive 完整。
- 验证证据：`go list -f` 边界检查如上；17 个命令目录同名主文件存在、无泛化主文件；`go test ./...` 无 FAIL；`go test -race ./...` exit 0；`go vet ./...`、`gofmt -l`、`git diff --check` clean；gopls 全仓无诊断；六个离线 shell 门禁全部 exit 0。
- 剩余风险：无需要追加修复 task 的代码问题；命令内 `writeMovieRows`/`writeJSON` 小写循环是 plan 设计的边界内重复，Task 11 删除 output 后由命令 owner + Line() 单一来源承载。
- 下一步：Task 10。

## Task 10：迁移个人状态、列表与排行命令

- 状态：`completed`
- 范围：迁移 watched、want、recent、collections、mark、unmark、rankings、top250、lists。
- 必须完成：
  - 每个顶层命令使用真实同名目录与主文件。
  - rankings 保留 movies/actors/playback；top250 保持独立；lists 保留 show/search/related。
  - 影片列表使用 movie；命名实体使用 entity；排行前缀和 lists 文本留在真实命令 owner。
  - 保持认证、mark/unmark 文案、JSON payload、generated_at、空结果和错误语义。
  - 为每个 owner 迁移或补充聚焦测试。
- 验证要求：相关命令与共享包 test/race/vet、gopls、help/输出契约、全仓测试。
- 实际做了什么：
  - 新增 `internal/cli/commands/{watched,want,recent,collections,mark,unmark}` 六个真实顶层命令（`commands/user` 的 Register 保持不变，root 仍注册旧路径；Task 11 切换）。
  - `commands/rankings` 重构为 `{rankings.go,movies.go,actors.go,playback.go}`：`New` 组（movies/actors/playback 各自 `NewMovies/NewActors/NewPlayback`）；`NewTop250` 变为过渡 seam（`return top250cmd.New(...)`）保持旧 root 注册可编译。
  - 新增 `commands/top250/top250.go`（独立顶层命令，`# generated_at=` 到 stderr、`writeRanked` 写 `#rank\t` 前缀）；确认基线无 `--json` flag（main 分支的 #14 未合入本分支，probe 锁定）。
  - `commands/lists` 重构为 `{lists.go,show.go,search.go,related.go}`：`New` 组 + `NewShow/NewSearch/NewRelated`；`writeListRows` 写 `id\tname\tmovies\tprivacy\tviews`（同包共享，lists/search/related 三子命令使用）。
  - 渲染边界：watched/want/recent/rankings(movies/playback) 用 `movie.ProjectAll`+`Row.Line`；collections 用 `entity.ProjectNamedAll`+`NamedRow.Line`；rankings actors 用 entity 投影写 `id\tname`（忽略 count）；top250 用 movie.Project + `#rank` 前缀（排行前缀留在命令 owner）；mark/unmark 文案 `已标记…`/`已取消标记…`/`无标记可取消…` 逐字保持；lists JSON 含 `current_page`（jsonx.RawString）；所有 JSON 用 jsonx.MarshalLine。
  - 认证语义：watched/want/recent/collections/mark/unmark/top250 用 `app.WithAuthedClient`；lists（MyLists）用 `app.WithAuthedClient`；show/search/related 用 `app.NewClient`；错误文本保持。
  - owner 测试：watched（构造/help/空列表）、want（构造/help）、recent（构造/help）、collections（构造/缺参/空列表）、mark（构造/`specify exactly one of --watched or --want` 前置错误）、unmark（构造/缺参）、rankings（组 3 子命令、movies 无 --json、actors flags、writeMovies 空、writeNamedNoCount 字面）、top250（构造/flags 无 --json/help/writeRanked 字面含 `#3\t…`/空列表）、lists（组 3 子命令+flags、writeListRows 字面、空列表、show help）。
  - 同步架构门禁：required_dirs 增加 `internal/cli/commands/{watched,want,recent,collections,mark,unmark,top250}`。
- 验证证据：`go test ./...` 无 FAIL/panic；`go test ./internal/cli/commands/...` 全 ok；`go test -race ./internal/cli/commands/...` 无 FAIL/DATA RACE；`go vet ./...` exit 0；`rg --files -g '*.go' -0 | xargs -0 gopls check` 全仓无诊断；`gofmt -l internal/cli` 清空；`git diff --check` exit 0；`sh scripts/test-architecture.sh` 与 `sh scripts/test-documentation.sh` exit 0；`build/javdb top250 --help` 与 `rankings movies --help` 无 `--json`、root help 契约测试通过。未运行真实凭据、真实 API、在线更新或外部写入。
- 剩余风险：
  - `commands/rankings.NewTop250` 是过渡 seam（委托 top250 包），Task 11 切换 root 后删除并改直接注册 `top250cmd.New`。
  - `commands/user` 旧 Register 仍在 root 注册（watched/want/recent/collections/mark/unmark 现由新包提供同行为命令），Task 11 删除 user 目录。
  - 各命令包内 `writeMovies`/`writeJSON` 小写循环延续 Task 8/9 的 plan 边界设计；排行前缀只在 top250 owner。
- 下一步：Task 11。

## Task 11：切换最终 CLI root 并删除旧路径

- 状态：`completed`
- 范围：让 `internal/cli/root.go` 直接组装最终命令树，迁移测试 owner，删除 facade/input/output/root 与旧分组目录。
- 必须完成：
  - 保持原 AddCommand 顺序、persistent flags、Run 错误输出和退出码。
  - 根 package 只保留 root/Run 与根契约测试，不保留渲染/输入 wrapper。
  - prompt 已由 auth owner；所有 output 函数已由 common、movie、magnet、entity 或真实命令替代。
  - 删除 cli/root、cli/facade、cli/input、cli/output、catalog、user、account。
  - gopls references 与 `rg` 确认无旧引用、无 command-to-command 循环。
  - 将架构门禁切换到最终 CLI allowlist 和同名文件规则。
- 验证要求：CLI 全包 test/race/vet、gopls、根 help/命令顺序、全仓测试、构建和架构门禁。
- 实际做了什么：
  - `internal/cli/root.go` 重写为真实根命令（`New`+`Run`），直接组装 26 个最终命令（auth,config,search,detail,comments,magnets,download,tags,browse,actor,series,maker,director,code,list,watched,want,recent,collections,mark,unmark,rankings,top250,lists,update,version），AddCommand 顺序与旧 root/root.go 逐项一致；persistent `--proxy/--host`、Run 先 `process.CleanupPendingWindowsUpdate`、错误→stderr+exit 1 保持。
  - 删除 `internal/cli/root/`（含 root_test.go）、`internal/cli/facade.go`、`internal/cli/input/`、`internal/cli/output/`、`internal/cli/commands/{account,catalog,user}`。`internal/cli` 根包现在只有 root.go + 根契约/help 测试。
  - 替换遗留 output 依赖：`commands/auth/check.go`、`commands/version/version.go`、`commands/update/update.go` 的 `output.EmitJSON` → `jsonx.MarshalLine`+写；`output.PrintUpdateResult` 移入 `commands/update` 本地 `printUpdateResult`（文本逐字）。
  - 测试 owner 迁移：`authclient_test.go` 的 withOptionalAuthClient 行为测试迁至 `internal/cli/app/auth_test.go`（isolateHome/seedAuth 一并迁移，直接调 `app.WithOptionalAuthClient`）；`print_test.go`（FilterHasMagnets/SearchTypeKey）删除（已由 movie/search owner 测试覆盖）；`search_cmd_test.go` 的 renderSearch 测试删除，改用 httptest 在 `commands/search/search_test.go` 补 `TestExecuteMoviesJSONHasMagnetsFilter`（--json --has-magnets 过滤）与 `TestExecuteNamedText`（--type actor 命名文本）；`detail/lists/comments/magnets_cmd_test.go` 中 PrintDetail/PrintLists/PrintMovieComments/parseSizeMiB 的 facade 断言删除（已由对应命令 owner 测试覆盖），保留全部 Run-based help 测试。
  - `contract_test.go` 的 `newRoot` → `New`（root 包删除后直接用真实根）。
  - 架构门禁切换最终 CLI allowlist：required_dirs 精确列出 26 个命令目录（去掉 root/input/output/account/catalog/user）；新增 `internal/cli/root.go` 存在、`facade.go`/`root/`/`input/`/`output/`/`commands/{account,catalog,user}` 不存在、每个命令目录同名主文件存在、禁止 command.go/output.go/printer.go/render.go 泛化主文件。
- 验证证据：`go build ./...` exit 0；`go test ./...` 无 FAIL/panic；`go test -race ./...` exit 0（无 FAIL/DATA RACE/panic）；`go vet ./...` exit 0；`rg --files -g '*.go' -0 | xargs -0 gopls check` 全仓无诊断；`gofmt -l` 全仓清空；`git diff --check` exit 0；六个离线 shell 门禁全部 exit 0；`rg` 确认 `internal/cli/(output|input|catalog|commands/account|commands/user)"` 零引用、旧路径不存在；`build/javdb --help` 根 help 与契约字面一致、`version --json` 正常；根 help/命令顺序契约测试通过。未运行真实凭据、真实 API、在线更新或外部写入。
- 剩余风险：
  - 维护者文档（architecture.md/development.md）仍描述旧的 CLI root/app/input/output/catalog 结构，Task 13 统一更新。
  - `scripts/releasenotes/compat.go` 与 root 门禁的 facade 项（compat）仍是过渡 seam，Task 12 处理。
  - `commands/rankings.NewTop250` 过渡 seam（委托 top250 包）在 root 切换后已不再被调用方使用，但函数仍在包内；Task 13 复查可一并移除（本 task 保持最小改动未删）。
- 下一步：Task 12。

## Task 12：删除 release-note compat 并迁移测试 owner

- 状态：`completed`
- 范围：删除 package-main 兼容 seam，把行为测试放回 audit/document/github/history/prepare。
- 必须完成：
  - main 只保留入口分派、help、未知命令测试。
  - 按 plan 固定 owner 迁移 validate/render、GitHub、contributor、prepare、coverage、bump、history 测试。
  - 删除 compat 并确认无根测试依赖旧 alias/forwarder。
  - 保持全部子命令、JSON、文本和 apply 边界。
  - 同步架构门禁。
- 验证要求：scripts 全包 test/race/vet、gopls、test-releasenotes、全仓测试和架构门禁。
- 实际做了什么：
  - `scripts/releasenotes/main.go` 保持纯分派（validate/audit/prepare/render/pr-validate/sync-history → document/audit/prepare/history），无业务逻辑。
  - 删除 `scripts/releasenotes/compat.go`（类型 alias、githubClient struct seam、syncHistoryConfig、全部 forwarder）。
  - 迁移行为测试到领域 owner：
    - `document/document_test.go`（package document）：ValidateReleaseDirectory 4 例（双语匹配/缺 source/双语 source 集不一致/初始 release commit）、RunRender 双语正文、ParseReleaseNoteDeclaration 3 例、RecommendedVersionBump 表格、RenderSourceLink、ValidateSourceCoverage。
    - `github/github_test.go`（package github）：PullRequestsForCommit→PullRequest→FirstMergedPullRequest httptest 全流程、ValidRepository。contributor 判定断言改为在 client 测试内验证 User.Type/Login（避免 github→audit 测试环）。
    - `audit/audit_test.go`（package audit）：IsExternalContributor owner/bot 排除、ValidatePullRequestEvent。
    - `prepare/prepare_test.go`（package prepare）：PrepareRelease 双语 notes+index（含 document.ValidateReleaseDirectory 复检）、ValidatePreparePlanMetadata version 不匹配、ValidatePreparePlan repeated source、ValidatePlanCoverage missing contributor。
    - `history/history_test.go`（package history）：SyncHistoricalRelease dry-run 不打补丁 + apply 打一次且保留 assets、创建缺失历史 Release 且不造 assets。
  - `main_test.go` 重写为仅入口分派：无参数→subcommand required、help/-h/--help→usage、未知子命令、六个子命令各自被正确分派（无参返回子命令自己的 flag/参数错误而非 unknown）。
  - 架构门禁：移除 facade_file 循环，新增 `compat.go` 必须不存在 + `main.go` 必须存在。
- 验证证据：`go test ./scripts/...` 全 ok（audit/document/github/history/prepare/releasenotes）；`go test -race ./scripts/...` 无 FAIL/DATA RACE；`go vet ./...` exit 0；`rg --files -g '*.go' -0 | xargs -0 gopls check` 全仓无诊断；`gofmt -l scripts` 清空；`git diff --check` exit 0；`sh scripts/test-architecture.sh` 与 `sh scripts/test-releasenotes.sh` 及其余离线门禁 exit 0；`rg` 确认无 compat 符号/`compat.go` 残留（document.go 的 `releaseNotePattern`/`releaseNoteCategories` 为领域包自身正则，非 compat）。未运行 GitHub API、真实 token、在线发布或 apply 外部写入。
- 剩余风险：
  - github_test 内联了 contributor 判定断言（User.Type/Login）而非调用 audit.IsExternalContributor（避免测试环）；audit_test 才是 IsExternalContributor 的 owner。
  - 维护者文档仍在描述 release-note compat seam，Task 13 统一更新。
- 下一步：集中检查-debug 4。

## 集中检查-debug 4：最终代码结构与 release-note

- 状态：`completed`
- 检查范围：全仓旧路径、死代码、命令目录、App API、config/update、release-note、公开契约、安全、包图、LSP、测试和文档缺口。
- 必须完成：发现问题时在 Task 13 前追加修复 task；不得用文档掩盖结构或行为偏差。
- 实际做了什么：
  - 全仓旧路径：`rg`+`test -e` 确认 `internal/cli/facade.go`、`internal/cli/{root,input,output}`、`internal/cli/commands/{account,catalog,user}`、`internal/shared`、`internal/javdb/appapi/facade.go`、`internal/update/{facade.go,types.go}`、`scripts/releasenotes/compat.go`、`scripts/internal/releasenotesrender` 全部不存在；`internal/config` 仅存 paths/settings 子目录、根 0 Go 文件。
  - 死代码：发现并移除 `commands/rankings.NewTop250` 过渡 seam（root 已直接注册 top250 包，该函数无调用方）；`transport.go` 过时 "public facade" 注释改为 model/SDK 身份说明。
  - 命令目录：17+9 命令目录同名主文件齐全，无 command.go/output.go/printer.go/render.go；架构门禁最终 allowlist 通过。
  - 命令依赖：`go list -f` 确认命令包间唯一跨命令导入已消失（rankings→top250 移除后无 command-to-command 依赖）。
  - App API/config/update/release-note：根 appapi 为真实 Client 组合（无手写 forwarder）；config 根无包；update 根只有 coordinator/interfaces；release-note main 纯分派。
  - 公开契约：SDK `go doc -all` 无 internal 泄漏；root help/命令顺序/输出字面契约测试全部通过。
  - 安全：无新增 secret 输出；命令远程操作全部经 sdk。
  - LSP/测试：全仓 gopls 无诊断；`go test ./...`、`go test -race ./...`、`go vet ./...`、build 全绿；六个离线门禁全部 exit 0。
  - 文档缺口：`docs/maintainers/{architecture,development}.md` 仍描述旧的 facade/catalog/account/shared-values/appapi-facade/compat 结构，需 Task 13 更新（本检查不覆盖文档）。
- 验证证据：旧路径不存在、NewTop250 删除后 `go build ./...` 与 `go test ./internal/cli/...` 通过；`go test -race ./...` exit 0；gopls/gofmt/git diff clean；六个离线门禁 exit 0；SDK doc 无泄漏；命令包无跨命令依赖。
- 剩余风险：无需要追加修复 task 的结构问题；仅保留 Task 13 的维护者文档与 ADR 更新义务。
- 下一步：Task 13。

## Task 13：更新维护者文档、ADR 与最终架构门禁

- 状态：`completed`
- 范围：准确记录最终目录、依赖方向、内部兼容策略和 machine-checkable 规则。
- 必须完成：
  - 更新 architecture、development、ADR 的 CLI/common/App API/config/update/release-note 描述。
  - 删除内部 facade/compat 是既定边界的旧表述，说明 API() 是公开 SDK 冻结例外。
  - 架构门禁实现最终命令 allowlist、同名文件、旧路径、依赖方向、无循环和 SDK internal allowlist。
  - 不修改 public docs、README、skill 或 changelog；若门禁发现其需要修改，先确认是否发生公开契约偏离。
- 验证要求：documentation/architecture tests、shell syntax、`go list`、`go doc`、gopls、全仓测试和 diff check。
- 实际做了什么：
  - 重写 `docs/maintainers/architecture.md`：总体流程改为 `cli/root.go → commands/*` 直接组装、`cli/{movie,magnet,entity}` 投影、`internal/common/{jsonx,scalar}`；App API 根包改为"真实 Client 组合层"描述（未导出别名嵌入+method promotion，非转发 facade）；config 根无包、update 根只 coordinator/interfaces、release-note main 仅分派；新增"内部兼容 facade/compat 是已删除的过渡产物"段落，明确 `Client.API()` 与 taxonomy `*tags.Doc` 是冻结例外。
  - 更新 `docs/maintainers/development.md` 目录地图：移除 root/input/output/catalog/account/user/shared-values 条目，替换为 26 个命令域、movie/magnet/entity、common 两子包、appapi 真实组合层、config/update 根无 facade；边界段落同步。
  - 重写 ADR 0001：标题改为"公开 SDK 与 JavDB 领域目录"，决策/后果/依赖约束全面对齐最终结构，明确"App API 根 Client 保留唯一原因是 API() 返回该类型，因此必须是真实组合层"。
  - 架构门禁已在 Task 11/12 落地最终 allowlist/同名文件/旧路径/依赖方向/无循环/SDK internal allowlist（本 task 复核通过）；`go doc -all sdk` 无泄漏。
  - 未修改 public docs、README、skill、changelog（`test-documentation.sh` 通过，说明公开契约无偏离）。
- 验证证据：`go test ./...`、`go test -race ./...`、`go vet ./...`、`go build ./...` 全绿；`sh -n scripts/test-architecture.sh` + `sh scripts/test-architecture.sh` exit 0；`sh scripts/test-documentation.sh` 与其余四个离线门禁 exit 0；gopls/gofmt/git diff clean；`rg` 确认 maintainer docs 中仅存的 "facade/compat" 表述均为"已删除"的说明性内容。
- 剩余风险：无。
- 下一步：Task 14。

## Task 14：执行完整离线回归并修复本目标问题

- 状态：`completed`
- 范围：运行 plan 固定的全仓 LSP、test、race、vet、build、二进制冒烟和所有离线脚本。
- 必须完成：
  - 只修复本目标造成的失败，不扩展相邻路线图。
  - 记录所有命令、exit code 和环境差异。
  - 检查构建产物、临时目录、格式和工作树状态。
  - 不运行真实 API、凭据、在线更新、发布或 apply。
- 验证要求：plan 第 12 节完整最终矩阵全部通过。
- 实际做了什么：
  - 按 plan §12 最终矩阵逐项执行：全仓 gopls（rg 文件流）、`go test ./...`、`go test -race ./...`、`go vet ./...`、`sh scripts/build.sh`、`build/javdb --help`、`build/javdb version --json`、六个离线脚本门禁、`go test ./scripts/login_probe.go`、`gofmt -l`、`git diff --check`，全部 exit 0。
  - 本 task 未发现需要修复的代码回归；未新增限制、网络调用、凭据读取或外部写入。
  - 检查构建产物（build/javdb 为既有忽略产物）、临时目录（无 _probe 等残留）、格式（gofmt 全仓清空）与工作树状态（用户文件保留、goal 文件完整）。
- 验证证据：
  - gopls 全仓无诊断（exit 0）；`go test ./...` 无 FAIL/panic；`go test -race ./...` exit 0 无 FAIL/DATA RACE/panic；`go vet ./...` exit 0。
  - `sh scripts/build.sh` 产出 build/javdb（version=dev commit=3da5d6f）；`build/javdb --help` exit 0；`build/javdb version --json` 输出 `{"build_date":...,"commit":"3da5d6f","version":"dev"}`。
  - `scripts/test-{architecture,documentation,releasenotes,package-release,homebrew-formula,workflows}.sh` 全部 exit 0；`go test ./scripts/login_probe.go` 编译通过；`gofmt -l $(rg --files -g '*.go')` 清空；`git diff --check` exit 0。
  - 未运行真实凭据、真实 API、在线更新、发布或任何 `--apply` 外部写入路径。
- 剩余风险：无已知构建、类型、测试或离线脚本失败。
- 下一步：Task 15。

## Task 15：终审与交付状态整理

- 状态：`completed`
- 范围：逐项核对 input、plan、公开契约、最终目录、错误处理、安全、测试、构建、文档和用户工作树。
- 必须完成：
  - 确认所有普通 task 和集中检查均 completed，或准确记录无法完成的阻塞项。
  - 确认无旧路径、无 facade/compat 残留、无未测试的新抽象。
  - 确认 SDK/CLI 契约测试与全部最终验证证据有效。
  - 确认未创建 commit、分支、push、release，未丢失用户改动。
  - 无已知高风险问题时才标记 goal 完成。
- 实际做了什么：
  - 逐项核对：`grep '状态：\`completed\`' goal-1/tasks.md` 为 18（Task 1–14 + 集中检查 1–4），本 task 完成后为 19；无 pending、无阻塞项。
  - input 逐条验收：以 dirty worktree 为唯一基线（HEAD 仍为 3da5d6f、origin/main 未动）；内部 import path 与兼容入口允许破坏——CLI facade/input/output/root、config 根 facade、update facade/根 alias、appapi facade、releasenotes compat、shared/values 全部删除；公开 SDK 不变——`Client.API()` 返回真实组合层 appapi Client，原 42 方法集可编译可调用（sdk/facade_test.go 方法值断言 + appapi/client_test.go），taxonomy `*tags.Doc` 返回类型冻结；App API 用 endpoint/* capability service（固定构造顺序），根 Client 是真实组合层（未导出别名嵌入 + method promotion，无手写 forwarder）；commands/ 只放真实命令、主文件与目录同名；六实体命令各自真实 Cobra 定义、共享 entity.Execute；cli/movie 严格为影片纯投影；cli/magnet 为磁力纯投影（detail --magnets 与 magnets 共用）；JSON 字节契约由 internal/common/jsonx 的 MarshalLine 统一保证；本轮未 commit/branch/push/release、未运行真实凭据/API。
  - 最终目录：`go list ./...` 70 包可加载无循环；26 个命令目录 + movie/magnet/entity + common 两子包 + appapi 组合层 + config/update/storage/release-note 领域包；旧路径全仓 `test -e`/`rg` 零命中；无 command-to-command 依赖；无未测试的新抽象（movie/magnet/entity Line()、entity.Execute 单页/全页/降级/错误路径、26 个命令 owner 测试、appapi client 构造/method-set/错误/helper、update coordinator 4 用例、release-note 6 领域测试）。
  - 公开契约：sdk external contract + facade_test（含 API() 方法集、错误匹配、ActorPeriod 别名）全 PASS；CLI root help/命令顺序/persistent flags/无网络参数错误字面契约全 PASS；输出字面契约（EmitJSON 单尾随换行 + HTML 不转义、空结果文案、磁力/评论/列表/排行格式）通过。
  - 安全：无新增 secret 输出；认证/凭据只由 auth 命令与 app.WithOptionalAuthClient/WithAuthedClient 处理；未运行真实 API/凭据/在线更新/`--apply`。
  - 文档：architecture/development/ADR 已对齐最终结构；未改 public docs/README/skill/changelog（test-documentation 通过，公开契约无偏离）。
  - 工作树：用户既有改动（README 双语、changelog、docs en/zh-CN sdk+cli-reference、sdk/rankings.go、skill、.idea/）全部保留；goal-1 三文件 + archive 完整。
  - 未创建 commit（HEAD 未变）、分支、push 或 release。
- 验证证据：`go test ./...` 59 包全 ok、`go test -race ./...` exit 0、`go vet ./...` exit 0、`go build ./...` exit 0；全仓 gopls 无诊断、`gofmt -l` 清空、`git diff --check` exit 0；六个离线脚本门禁全部 exit 0；`go test ./scripts/login_probe.go` 编译通过；`build/javdb --help` 与 `version --json` 冒烟正常；SDK 关键契约测试逐项 PASS（上方已列）；旧路径/残留/未测试抽象检查零命中。
- 剩余风险：无已知高风险问题。已确认无阻塞项。
- 下一步：无；goal 可标记完成。
