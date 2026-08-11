# Goal 1 任务清单

执行规则：每轮只处理第一个 `pending` task；完成后填写“实际做了什么、验证证据、剩余风险、下一步”，再进入下一轮。除维护者明确要求外不自动 commit。

## Task 1：建立 shared values 包与边界表格测试

- 状态：`completed`
- 范围：确认现有 String/Int64/ObjectArray/ObjectSlice 的调用方与语义；新增 `internal/shared/values` 及表格测试；保留 CLI/App API 的领域差异。
- 实际做了什么：核对了 `internal/cli/print.go` 的 `anyString`/`anyToInt`、`internal/javdb/appapi/search.go` 的 `decodeObjectArray` 与 `internal/javdb/appapi/browse.go` 的 `asMapSlice`。新增 `internal/shared/values/values.go`，提供 `String`、`Int64`、`ObjectArray`、`ObjectSlice`；新增覆盖 nil、JSON number、Go 整数/浮点数、字符串、溢出、非法 JSON、null 元素、map slice、`[]any` 过滤和 JSON fallback 的表格测试。未迁移 CLI/App API 的领域 wrapper，未改变现有调用方行为。
- 验证证据：`gopls check` 对仓库全部 Go 文件无输出；`go test ./internal/shared/values` 通过；`go test -race ./internal/shared/values` 通过；`go vet ./internal/shared/values` 通过；`go test ./...` 全部通过；`gofmt` 后无差异。直接执行 `gopls check ./...` 会因 gopls v0.21.1 将 `./...` 解析为文件路径而报错，改用 `rg --files -g '*.go' -0 | xargs -0 gopls check` 完成等价全文件诊断。
- 剩余风险：shared helper 尚未接入领域包；后续 App API/CLI 迁移时必须逐个对比 wrapper 的浮点、前缀数字和 truthy 语义，避免把领域差异误合并。
- 下一步：Task 2 迁移 App API 的 model/client/codec/media/endpoint 实现。

## Task 2：迁移 App API 的 model/client/codec/media/endpoint 实现

- 状态：`completed`
- 范围：按 capability 拆分内部实现，控制依赖方向，保留请求、响应、认证、媒体/HLS 行为。
- 实际做了什么：将 App API 实现从根目录迁移到 `client`、`model`、`codec`、`media` 与 `endpoint/{auth,browse,entity,lists,magnets,movie,rankings,search,user}`。`client` 保留签名 HTTP、公共参数、token/device/lang、envelope、状态码和认证错误映射，并以 `FetchMedia` callback 接入媒体包；`model` 保留 Options、SearchResult、错误类型和映射；`codec` 抽取 RawMessage、object array/slice、JWT 与用户 ID 解码；各 endpoint 改为接收 `*client.Client` 的 adapter。媒体图片/XOR、HLS playlist/key/IV/PKCS#7 与独占文件写入逻辑随测试迁入 `media`。保留 CLI 浮点/磁力前缀数字/truthy 等领域 wrapper。新增 codec 表格/身份测试和 client envelope 错误映射测试。
- 验证证据：`gofmt` 完成；`gopls check` 对新迁移包无诊断；`go test -race ./internal/javdb/appapi/client ./internal/javdb/appapi/model ./internal/javdb/appapi/codec ./internal/javdb/appapi/media ./internal/javdb/appapi/endpoint/...` 通过；对应 `go vet` 通过；各迁移 endpoint/client/codec/media 测试通过。直接运行 `go test ./...` 当前只因根 `internal/javdb/appapi` facade 尚未在 Task 3 恢复而失败，失败证据为 SDK/CLI 报 “no required module provides package .../internal/javdb/appapi”，未发现迁移包自身编译失败。
- 剩余风险：根包当前暂时没有 facade，SDK、CLI、`scripts/login_probe.go` 尚未重新接通；公开类型 alias、方法转发、媒体公开入口和全仓 LSP/测试须由 Task 3 完成验证。当前尚未创建 commit，保留用户工作树改动。
- 下一步：Task 3 恢复 App API 根 facade，并验证 SDK 与 `scripts/login_probe` 源码兼容。

## Task 3：恢复 App API 根 facade 并验证 SDK/login_probe 源码兼容

- 状态：`completed`
- 范围：根包 alias/forwarder、SDK 调用关系、错误类型/常量/请求类型/工具函数和 `scripts/login_probe.go` 兼容性。
- 实际做了什么：新增 `internal/javdb/appapi/facade.go`，恢复原根包的常量、变量、Options/SearchResult/Error/AuthRequired/LoginResponse alias、New、LoadOrCreateDeviceUUID、纯 helper 和完整 Client 方法集。Client 只持有 `*appapi/client.Client`，endpoint 与 media 通过 forwarder 接入；新增 `facade_test.go`，对稳定类型身份、构造器、变量、全部 Client 方法签名和关键 helper 行为做编译/行为断言。
- 验证证据：`go doc -all github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi` 核对导出符号与原基线清单；`go test ./...` 通过；`go test -race ./...` 通过；`go vet ./...` 通过；全仓 Go 文件 `gopls check` 无诊断；`go test ./scripts/login_probe.go` 通过；SDK、CLI 和 facade 定向测试通过。未执行真实 API、真实凭据或 login probe 主程序。
- 剩余风险：Task 1-3 后需要集中复查新目录边界、旧实现残留、架构门禁和维护者文档；当前尚未更新 `scripts/test-architecture.sh` 的新目录规则，也未创建 commit。
- 下一步：集中检查-debug 1，复查 Task 1-3 的需求偏离、架构边界、测试/LSP/构建与文档同步。

## 集中检查-debug 1（Task 1-3 后）

- 状态：`completed`
- 检查范围：需求偏离、重复/死代码、LSP/类型诊断、构建/测试、架构边界、安全与文档同步；发现问题追加修复 task。
- 实际做了什么：复查了目标文件要求、当前用户未提交文件、App API 新目录和 import 方向；确认根 facade 只转发到 `client/model/codec/media/endpoint`，CLI/SDK 未直连协议实现，旧根实现已不存在且没有重复 helper 被误合并。检查认证/媒体路径未新增 secret 输出、在线调用、timeout/retry/truncation/fallback；确认 README、CLI/SDK 用户文档和 changelog 的既有用户改动均保留。维护者架构文档、ADR 与架构脚本的细化更新尚未做，但已由后续 Task 8/9 覆盖，不需追加重复修复 task。
- 验证证据：`go test ./...`、`go test -race ./...`、`go vet ./...`、全仓 Go 文件 `gopls check` 均通过；`go test ./scripts/login_probe.go` 通过；`sh scripts/test-architecture.sh`、`sh scripts/test-documentation.sh`、`sh scripts/test-package-release.sh`、`sh scripts/test-homebrew-formula.sh`、`sh scripts/test-workflows.sh` 均以 exit 0 完成；门禁脚本未留下临时目录或额外工作树改动。
- 剩余风险：维护者文档仍未描述新的 App API 子目录，架构脚本仍未检查新目录存在性、import cycle、CLI 协议直连和 SDK internal 类型边界；这些是计划内 Task 8/9 的交付项。当前未创建 commit。
- 下一步：Task 4 迁移 update 与 config 子包并恢复根 facade。

## Task 4：迁移 update 与 config 子包并恢复根 facade

- 状态：`completed`
- 范围：`update/model|archive|release|source|process`、`config/paths|settings`；保持更新来源识别、校验替换、配置优先级与根包公开 API。
- 实际做了什么：将 update 实现与测试按职责迁移到 `model`、`archive`、`release`、`source`、`process`；model 保存安装来源、release asset、Request/Result 与更新常量，archive 保留校验和、归档解包、暂存替换流程，release 独立 GitHub Releases、SemVer 与代理 HTTP client，source 独立安装来源识别，process 独立命令执行、候选二进制校验、平台替换与可执行文件路径解析。根 `internal/update` 以 alias/forwarder 恢复原类型、构造器与函数，Coordinator 继续负责策略组装。config 迁移为 `paths` 与 `settings`，根 `internal/config` 保留路径、schema、默认值、优先级解析与校验的兼容 facade；新增两个 facade 类型/行为断言测试。未改变用户文档、SDK 或 changelog 改动。
- 验证证据：全仓 Go 文件 `gopls check` 无诊断（包含当前平台和 gopls 报告的 windows 变体）；`go test ./internal/update/... ./internal/config/...`、`go test -race ./internal/update/... ./internal/config/...`、`go vet ./internal/update/... ./internal/config/...`、`go test ./...`、`go vet ./...` 均通过；`sh scripts/test-architecture.sh` 与 `git diff --check` 通过；`go doc -all` 核对根 `internal/update`、`internal/config` 的兼容导出符号。未执行真实凭据、真实 API 或在线更新写入路径。
- 剩余风险：process/model 子包没有额外独立的历史测试文件，但候选校验与替换仍由 archive 测试覆盖，root facade 断言了接口/类型身份；维护者文档与更严格架构门禁仍留待 Task 8/9。当前未创建 commit。
- 下一步：Task 5 按职责整理 `internal/storage/auth` 与 `internal/storage/tags` 文件，不制造空子包。

## Task 5：按职责整理 storage/auth 与 storage/tags 文件

- 状态：`completed`
- 范围：只拆 model/store/file/resolve 等职责文件，不制造空子包；保持 JSON、权限、缓存格式和调用行为。
- 实际做了什么：将 `internal/storage/auth` 拆为 `model.go`（Account/Store/schema）、`store.go`（Upsert/Remove/Use/UpdateToken）、`resolve.go`（Default/Get）和 `file.go`（Load/Save/FileStore/Open/Commit）；将 `internal/storage/tags` 拆为 `model.go`（Doc/Category/Tag）、`file.go`（Path/Load/Save）和 `resolve.go`（AliasMap/ResolveRefs）。保留原包路径、导出类型/方法、JSON tags、认证临时文件 + rename、0600 权限、tag cache 0700 目录与 0644 文件权限，以及缺失/无 categories 的 nil 语义。新增认证与 taxonomy 缺失/非法 JSON 边界测试；未制造空子包，也未触碰用户已有文件。
- 验证证据：storage 全部 Go 文件 `gopls check` 无诊断；`go test ./internal/storage/...`、`go test -race ./internal/storage/...`、`go vet ./internal/storage/...`、`go test ./...`、`go vet ./...` 均通过；`gofmt -d internal/storage/auth internal/storage/tags` 无输出；`git diff --check` 通过；`go doc -all` 核对 `internal/storage/auth` 与 `internal/storage/tags` 的导出符号。未执行真实凭据或外部状态写入。
- 剩余风险：本 task 只整理已有单包职责，未新增 storage facade 或子包；后续 CLI 拆分必须继续从公开 SDK 与现有 storage 包调用，避免把认证 token 输出到用户可见日志。当前未创建 commit。
- 下一步：Task 6 按 `cli/app`、`cli/root`、`cli/commands/*`、`cli/input`、`cli/output` 拆分 CLI，并保留根 wrapper 与命令输出契约。

## Task 6：迁移 CLI app/root/commands/input/output 并保留根 wrapper

- 状态：`completed`
- 范围：Cobra 树、IO/runtime/client/auth 依赖、命令域、输入和用户可见渲染；只经公开 SDK facade，保持命令/flag/text/JSON 行为。
- 实际做了什么：新增 `internal/cli/app` 集中管理 IO、root flags、config runtime、SDK client、认证 store、可选/强制认证重试和 update coordinator；新增 `cli/root` 复刻原 persistent flags 与 AddCommand 顺序；将命令迁移至 `commands/{account,config,catalog,lists,rankings,user,update,version}`，将交互输入迁移至 `input`，将文本/JSON/movie/detail/comment/list/ranking/update 渲染迁移至 `output`。根 `internal/cli` 仅保留 `Run` 与原有导出渲染/输入函数的薄 wrapper，并保留旧 in-package 测试所需的私有 compatibility shim。新增 root command tree 与 output raw-field 聚焦测试；未修改用户 CLI reference、README 或 changelog。
- 验证证据：CLI 全部 Go 文件 `gopls check` 无诊断；`go test ./internal/cli/...`、`go test -race ./internal/cli/...`、`go vet ./internal/cli/...`、`go test ./...`、`go vet ./...` 均通过；`sh scripts/test-architecture.sh`、`gofmt -d internal/cli` 与 `git diff --check` 通过；`go doc -all internal/cli` 核对原导出函数仍存在；新 root 测试核对 `--proxy`、`--host` 和全部顶层命令；`rg` 确认 `internal/cli` 未导入 `internal/javdb/appapi` 或 `internal/javdb/protocol`。未执行真实凭据、真实 API 或外部写入路径。
- 剩余风险：旧 CLI 行为测试继续位于根 facade 包中，以覆盖源码兼容入口；新领域包已通过根命令路径和新增聚焦测试覆盖。更严格的 CLI 依赖/循环架构规则与第二个集中检查留待后续 Task 9/Checkpoint 2；当前未创建 commit。
- 下一步：集中检查-debug 2，复查 Task 4-6 的结构、公共契约、输出/交互、LSP、测试、架构边界、安全与文档同步。

## 集中检查-debug 2（Task 4-6 后）

- 状态：`completed`
- 检查范围：需求偏离、根 facade 与调用方、CLI 交互/输出、LSP/类型诊断、构建/测试、架构边界、安全与文档同步；发现问题追加修复 task。
- 实际做了什么：复核了目标文件与当前任务状态，确认 Task 4–6 的目录布局、根 facade、CLI 命令注册和依赖方向与目标一致；`go list` 显示 CLI 命令域只依赖 `sdk` 执行远程操作，App API endpoint 依赖 client/model/codec/media，update Coordinator 负责组装。检查旧根目录后确认实现已移入子包，根目录只剩兼容 facade、Run 入口和原有根包测试。通过 `go run ./cmd/javdb --help` 与 `go run ./cmd/javdb version --json` 做离线 CLI 冒烟，确认 persistent `--host/--proxy`、顶层命令树和 version JSON 可用。检查认证、token、密码、媒体和分页相关代码，未发现本轮新增 secret 输出或无证据的 timeout、重试、截断、静默 fallback；现有兼容语义（包括认证失败匿名重试、单页评论和已有分页上限）保持原实现。工作树中用户已有 README、双语 CLI/SDK 文档、架构文档、changelog、`sdk/rankings.go`、skill 和 `.idea/` 改动均仍在，未执行 reset/checkout/commit。
- 验证证据：全仓 Go 文件通过 `gopls check`（无诊断）；`git diff --check` 通过；`go test ./...`、`go test -race ./...`、`go vet ./...`、`sh scripts/build.sh` 均通过；`sh scripts/test-architecture.sh`、`sh scripts/test-documentation.sh`、`sh scripts/test-package-release.sh`、`sh scripts/test-homebrew-formula.sh`、`sh scripts/test-workflows.sh` 均 exit 0；`go doc -all` 核对 `sdk`、`internal/javdb/appapi`、`internal/cli` 的兼容导出面。未运行真实凭据、真实 API、在线更新、发布或外部写入。
- 剩余风险：维护者 `architecture.md`/`development.md`/ADR 尚未具体描述新的 appapi、CLI、update/config 子目录，`scripts/test-architecture.sh` 尚未增加新目录存在性、import cycle、CLI 协议直连和 SDK internal 类型检查；这两项已由 Task 8/9 覆盖。没有发现需要额外插入的修复 task。
- 下一步：进入 Task 7，迁移 `scripts/releasenotes` 核心逻辑到 `scripts/internal/releasenotes/{model,github,audit,document,prepare,history}`，保持命令入口和行为兼容。

## Task 7：迁移 release-note 脚本核心逻辑

- 状态：`completed`
- 范围：保留 `scripts/releasenotes` 命令入口，将核心逻辑拆入 `scripts/internal/releasenotes/{model,github,audit,document,prepare,history}`；不改 changescope/login_probe 的结构。
- 实际做了什么：新增 `scripts/internal/releasenotes/model` 的 release-note、GitHub、audit、prepare 和 Release 数据模型；新增 `github` 的 REST client、PR/Release 读取写入和 HTTP 错误边界；新增 `audit` 的审计收集、PR 事件校验、贡献者识别和 JSON 报告；新增 `document` 的双语 changelog 校验、来源链接、release-note 声明解析、版本 bump、prepare 文档渲染和 GitHub Release 正文渲染；新增 `prepare` 的计划校验、覆盖检查、文件生成与显式 apply；新增 `history` 的历史 Release dry-run/apply 同步。`scripts/releasenotes/main.go` 现在只保留命令入口/分派，`compat.go` 只保留原根测试 seam 的 alias/薄 forwarder；删除旧的 package-main 实现与重复的 `scripts/internal/releasenotesrender`。未改变 `scripts/changescope`、`scripts/login_probe.go` 或 release-note 命令的 flag、JSON、文本和 apply 边界。
- 验证证据：`go test ./scripts/...`、`go test -race ./scripts/...`、`go vet ./scripts/...`、全仓 `go test ./...`、全仓 `go vet ./...` 均通过；新脚本目录 Go 文件 `gopls check` 与全仓 Go 文件 `gopls check` 均无诊断；`sh scripts/test-releasenotes.sh`、`sh scripts/test-architecture.sh`、`sh scripts/test-documentation.sh` 和 `git diff --check` 均通过；离线运行 `go run ./scripts/releasenotes --help`、各子命令 help、现有 `main_test.go`、standalone `go test ./scripts/login_probe.go` 均验证入口/行为。检查 `go list` 确认新包依赖单向且无循环；`rg` 确认旧 `releasenotesrender` 无残留引用。未执行 GitHub API、真实 token、在线发布或 `sync-history --apply`。
- 剩余风险：现有行为测试仍位于 `scripts/releasenotes` 根 package，通过 compat wrapper 覆盖新包；后续 Task 8/9 将同步维护者目录文档并把 release-note 新目录纳入更严格架构门禁。当前未创建 commit，保留用户工作树改动。
- 下一步：Task 8 更新 `docs/maintainers/architecture.md`、`development.md` 与 ADR，准确记录新的 CLI/App API/update/config/storage/release-note 领域边界。

## Task 8：清理重复实现、补中文注释并更新维护者文档/ADR

- 状态：`completed`
- 范围：仅删除已由引用与测试证明安全的重复实现；同步 `architecture.md`、`development.md`、ADR 及必要注释，保留用户文档和 changelog 改动。
- 实际做了什么：更新 `docs/maintainers/architecture.md`，记录 CLI `root/app/commands/input/output`、App API `client/model/codec/media/endpoint/*`、config paths/settings、update 五个职责包、storage 文件职责、shared values 和 release-note 六个领域包，并明确 root facade、CLI 不直连协议、deprecated `Client.API()` 兼容例外及文档修改路由；保留了用户已有的 SDK ranking 说明。更新 `docs/maintainers/development.md` 的离线验证命令、gopls 版本兼容用法、目录地图、release-note dry-run/apply 边界和新领域依赖说明。修正 ADR 中过时的顶层 `javdb/` 表述，记录 `sdk/` 唯一公开 facade、各领域目录和依赖约束。将 release-note audit/history 重复的仓库标识校验集中到 `scripts/internal/releasenotes/github`，并补齐迁移 wrapper 的中文意图注释；未改 CLI/SDK public docs、README 或 changelog。
- 验证证据：`gofmt -d scripts/internal/releasenotes scripts/releasenotes` 无输出；全仓 Go 文件 `gopls check` 无诊断；`go test ./scripts/...`、`go test -race ./scripts/...`、`go vet ./scripts/...` 通过；`sh scripts/test-documentation.sh`、`git diff --check` 通过；`rg` 确认旧 `releasenotesrender` 和已删除旧路径无残留引用；`git status` 确认 README、双语 public docs、changelog、`.idea/` 等用户改动仍在，未执行 reset/checkout/commit。
- 剩余风险：`scripts/test-architecture.sh` 尚未把本轮新增 release-note 目录和更严格 import/SDK facade 规则固化为门禁，Task 9 处理；当前架构文档明确保留 deprecated `Client.API()` 这一既有兼容例外，后续门禁不能误删公开契约。
- 下一步：Task 9 更新 `scripts/test-architecture.sh`，增加新目录存在性、无 import cycle、CLI 协议直连和 SDK internal 暴露边界检查，并补齐必要聚焦测试。

## Task 9：增强架构门禁与兼容性测试

- 状态：`completed`
- 范围：更新 `scripts/test-architecture.sh` 的目录、循环依赖、CLI 协议直连、SDK internal 类型检查；补齐 facade、输出、endpoint 和共享 helper 的聚焦测试。
- 实际做了什么：扩展 `scripts/test-architecture.sh`，固定检查 CLI、App API、update、config、storage、shared values 和 release-note 领域目录及各根 facade，拒绝旧实现目录，执行 `go list ./...` 验证包图可加载且无 import cycle，检查 CLI 不直连 `internal/javdb/appapi`/protocol、SDK 不直连 App API 深层实现，并用 `go doc -all` 检查公开 SDK 文档不泄露 `internal`/协议实现类型；保留并核对已有 App API facade、CLI output、endpoint 和 shared values 测试，新增 `sdk/facade_test.go` 对 deprecated `Client.API()` 的返回类型、非空转发和 device UUID 行为做编译/行为断言。未改变 CLI/SDK 用户契约，也未执行网络请求。
- 验证证据：`sh -n scripts/test-architecture.sh`、`sh scripts/test-architecture.sh`、`go test ./sdk ./internal/shared/values ./internal/javdb/appapi/...`、`go test ./internal/cli/...`、对应聚焦 `go vet`、聚焦 `go test -race` 和 `git diff --check` 均通过；门禁中的 `go list` 与 `go doc` 检查实际执行成功。未使用真实凭据、真实 API 或外部写入。
- 剩余风险：完整全仓回归、构建和既有脚本集合仍待 Checkpoint 3/Task 10；SDK 的 `Client.API()` 是目标明确要求保留的 deprecated facade 例外，门禁只阻止深层实现泄露，不会误禁该兼容入口。当前未创建 commit。
- 下一步：进入集中检查-debug 3，复查 Task 7–9 的全仓结构、契约、文档、门禁和安全边界。

## 集中检查-debug 3（Task 7-9 后）

- 状态：`completed`
- 检查范围：全仓结构、公共契约、脚本/文档/架构门禁、安全、死代码、LSP、构建与测试；发现问题追加修复 task。
- 实际做了什么：以当前工作树审计了 Task 7–9 的新增/删除路径、Go 包图、根 facade 边界、release-note compat wrapper、SDK 导出面、用户既有改动和敏感信息路径。确认 root 目录只剩 facade/Coordinator/入口，App API 实现包不反向导入 root facade，协议实现只有 `appapi/client` 依赖，release-note 实现不依赖 CLI/SDK，shared values 生产代码只有四个目标导出 helper，未发现空目标包或旧实现引用。发现并修复架构门禁对迁移后残留空目录的误报：移除空的 `scripts/internal/releasenotesrender`；同时将 SDK internal 依赖收敛为既有 `config`、根 `appapi`、`storage/tags` 白名单，并增加 App API 反向依赖与 release-note→CLI/SDK 检查。为准确记录不可变公开契约，在维护者架构文档和 ADR 中注明 taxonomy `*tags.Doc` 与 deprecated `Client.API()` 是冻结兼容例外，新能力不得扩大例外。未发现需要追加修复 task。
- 验证证据：全仓 `gopls check`（`rg --files -g '*.go' -0 | xargs -0 gopls check`）无诊断；`go list ./...`、`go test ./...`、`go vet ./...` 均通过；`sh -n scripts/test-architecture.sh`、`sh scripts/test-architecture.sh`、`sh scripts/test-documentation.sh`、`sh scripts/test-releasenotes.sh` 和 `git diff --check` 均通过；SDK/App API/CLI/release-note 公开与兼容导出面通过 `go doc`/源码核对；`git status` 仍只包含 goal 预期的用户改动、领域迁移改动和 goal 文件，无冲突或测试生成物。未执行真实凭据、网络 API、在线发布或外部写入。
- 剩余风险：完整 race、构建和全部既有离线脚本集合仍由 Task 10 执行；当前 SDK 的两个 taxonomy 方法与 `Client.API()` 仍有基线 internal 类型/adapter 兼容暴露，但已明确冻结且由门禁阻止新增路径。当前未创建 commit。
- 下一步：Task 10 执行完整离线 LSP、测试/race/vet、构建和所有既有脚本回归，只修复本 goal 已发现的问题。

## Task 10：执行完整离线回归并修复已知问题

- 状态：`completed`
- 范围：按计划运行 gopls、Go 测试/race/vet、build 与所有既有离线检查；只修复本 goal 已发现的问题，不扩展路线图。
- 实际做了什么：按完整矩阵执行全仓 LSP、普通测试、race、vet、本机构建、构建产物离线冒烟和既有 shell 门禁；本 task 未发现需要修复的代码回归，也未新增限制、网络调用、凭据读取或外部写入。构建生成的 `build/javdb` 为既有忽略产物，fixture 脚本的临时目录均由脚本 trap 清理。
- 验证证据：`gopls v0.21.1` 直接执行 `gopls check ./...` 会将 `./...` 当作文件路径并报 `no such file or directory`；按维护者文档使用 `rg --files -g '*.go' -0 | xargs -0 gopls check`，全仓无诊断。`go test ./...`、`go test -race ./...`、`go vet ./...`、`sh scripts/build.sh` 均 exit 0；`build/javdb --help` 与 `build/javdb version --json` 离线冒烟通过；`sh scripts/test-architecture.sh`、`sh scripts/test-documentation.sh`、`sh scripts/test-releasenotes.sh`、`sh scripts/test-package-release.sh`、`sh scripts/test-homebrew-formula.sh`、`sh scripts/test-workflows.sh` 均 exit 0；`go test ./scripts/login_probe.go`、`gofmt -l $(rg --files -g '*.go')`、`git diff --check` 通过。未运行真实 API、真实凭据、在线发布或 `--apply` 外部写入路径。
- 剩余风险：无已知构建、类型、测试或离线脚本失败；仅保留 gopls 版本对 `./...` 参数的环境差异，已由文档化 fallback 覆盖。Task 11 仍需做最终 requirement-by-requirement 审计并整理交付状态。当前未创建 commit。
- 下一步：Task 11 进行终审，复核公开 SDK/CLI 契约、目录/门禁、文档、安全、回滚边界和最终工作树状态；确认无已知高风险问题后标记 goal 完成。

## Task 11：终审与交付状态整理

- 状态：`completed`
- 范围：从兼容性、代码质量、安全、错误处理、测试覆盖、构建、文档和回滚角度做最终复查；确认无已知高风险问题，完整回写状态。
- 实际做了什么：按目标文件逐项审计当前工作树：确认 `internal/shared/values` 仅提供四个目标 helper 及边界表格测试；CLI 已按 `root/app/commands/*/input/output` 分域且根 wrapper 保持入口/渲染兼容；App API 已按 `client/model/codec/media/endpoint/*` 分域且 root facade 只做 alias/forwarder；update/config/storage/release-note 均完成目标职责拆分并保留根包/命令 seam。复核 SDK、CLI、App API、update/config facade 的公开导出与方法签名，核对架构文档、development、ADR、架构门禁和用户既有改动；确认无新增无据 timeout、重试、截断、静默 fallback、secret 输出、在线发布或外部状态写入。未发现需要追加修复 task；未创建 commit，符合仓库提交门禁。
- 验证证据：当前 `git status --short` 保留用户列明的 README 双语文件、CLI/SDK 文档、changelog、`sdk/rankings.go`、skill 和 `.idea/`，无冲突状态或测试临时目录；`git diff --check` 通过。`go list ./...` 包图无错误；`go doc -all` 与 facade 编译/行为断言核对 SDK、App API、CLI、update/config 契约；全仓 gopls fallback、`go test ./...`、`go test -race ./...`、`go vet ./...`、`sh scripts/build.sh`、build 二进制 help/version 冒烟，以及 architecture/documentation/releasenotes/package-release/homebrew/workflows 六类脚本均已 exit 0。`go test ./scripts/login_probe.go`、全仓 gofmt 检查也通过。
- 剩余风险：无已知高风险问题。gopls v0.21.1 直接接受 `gopls check ./...` 的参数行为与其他版本不同，已由维护者文档记录的全文件 fallback 覆盖；基线的 taxonomy `*tags.Doc` 返回类型和 deprecated `Client.API()` 仍是冻结兼容例外，架构门禁禁止新增类似暴露。构建产物留在既有忽略路径，不纳入交付跟踪；提交、推送和发布仍由用户后续明确决定。
- 下一步：无未完成 goal task；由客户端标记 goal 完成并向用户交付结果。
