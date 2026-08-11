# Goal 2 任务清单

执行规则：

- 每轮完整读取 `input.md`、`plan.md`、`tasks.md`。
- 每轮只处理第一个 `pending` task，不合并、不跳序、不顺手处理相邻任务。
- 每三个普通 task 后执行一次集中检查-debug；发现问题时先追加精确修复 task。
- 代码 task 使用 TDD：先建立或迁移失败测试，再实现，再运行聚焦回归。
- 修改公开或跨包符号前使用 `gopls references` 核对调用方；修改后先读 LSP 诊断。
- 每个代码 task 完成前必须补齐 owner 测试并记录可复现验证证据。
- 使用现有 Go 1.26.4 工具链验证；默认 `go` 的 1.26.3/GOROOT 1.26.4 混用未修复前不得用于判定代码失败。
- 不自动安装依赖、commit、建分支、push、发布或运行真实凭据、真实 API、在线 update。
- 不回退、覆盖或删除与本目标无关的工作树内容。
- 本文件当前只初始化未来任务；全部 task 均未执行。

## Task 1：重新冻结实施基线

- 状态：`completed`
- 范围：记录实施开始时的 Git、工具链、包图、LSP、CLI 契约、测试分布和架构门禁。
- 必须完成：
  - 记录 HEAD、branch、remote refs 与 `git status --short`，明确用户既有文件。
  - 记录 `internal/cli` 文件树、根测试清单、包 import 图与 `app/movie/magnet/entity` 符号引用。
  - 用 `gopls references` 复核所有将改名或移动的导出符号调用方。
  - 使用 Go 1.26.4 运行当前 `go test ./internal/cli/...`、全仓测试、架构门禁和文档门禁。
  - 若基线失败，记录真因并停止，不把既有失败归因于本目标。
- 验证要求：`gopls check`、`go list ./...`、`go test ./...`、`sh scripts/test-architecture.sh`、`sh scripts/test-documentation.sh`、`git diff --check`。
- 实际做了什么：
  - Git 基线：`HEAD=9e8f965dce1f7153db540338849bea8a0d944463`（分支 `codex/domainization-remove-facades`），`origin/codex/domainization-remove-facades` 同 commit；remote 为 github.com/FlanChanXwO/javdb-cli.git。
  - Worktree：`git status --short` 仅 4 项——goal-2/{input,plan,tasks}.md（staged，/goal 初始化所建）与 `.idea/`（untracked，用户本地 IDE 目录）；无其他改动、无冲突。
  - 工具链：`which go` 现直接解析到 1.26.4 toolchain `/Users/flanchan/.local/share/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.4.darwin-arm64/bin/go`，`go version` = go1.26.4；input.md 记录的 1.26.3/GOROOT 1.26.4 混用在实施前已消除，plain `go` 可直接用于验证。gopls v0.21.1。
  - 包图：`go list ./...` 70 包可加载无循环；`internal/cli` 含根包、app、movie、magnet、entity 与 26 个命令包。
  - `internal/cli` 文件树：根目录 `root.go` + 13 个 `*_test.go`（comments_cmd/detail/download_cmd/entity_cmd/lists_cmd/magnets_cmd/rankings_cmd/search_cmd/tags_browse_cmd/update_cmd/user_cmd/version_cmd + contract_test）；`app/` 3 生产文件（app.go/auth.go/update.go）+ auth_test.go；`movie/`（movie.go/movie_test.go）；`magnet/`（magnet.go/magnet_test.go）；`entity/`（entity.go/entity_test.go）；`commands/*` 26 个真实命令目录。
  - 包 import 图：`app` 导入 config/paths、config/settings、storage/auth、update+子包、buildinfo、sdk、io；`movie` 仅 scalar+strconv；`magnet` 仅 fmt+scalar+strconv；`entity` 导入 movie+scalar+sdk。
  - 符号引用（gopls references + rg 交叉核对）：
    - `movie.FilterHasMagnets`：生产调用方 8 个命令（browse/rankings.movies/rankings.playback/recent/search/top250/want/watched）+ `entity/entity.go:84`；测试 `movie/movie_test.go` 2 处。
    - `entity.Execute`：actor/code/director/list/maker/series 6 个命令。
    - `entity.NamedRow/ProjectNamed/ProjectNamedAll`：entity.go 内部 + 各命名列表命令（collections/rankings.actors/search/list 等）消费 `ProjectNamedAll`。
    - `app.Flags`：root.go 与 26 个命令构造器/测试（gopls 列出的调用点覆盖全部 `commands/*`）；`app.IO/NewIO/LoadRuntime/NewClient/OpenAuth/WithAuthedClient/WithOptionalAuthClient/ResolveUpdateProxy/NewProductionUpdateCoordinator/BuildInfo` 调用方同 `app.Flags` 集合。
  - CLI 契约基线：`contract_test.go` 含根 help 全量字面、9 个关键命令组 help 字面、persistent flags、无网络参数错误（缺参/未知命令/update --json）；version/update/download/search/detail/comments/magnets/entity/lists/rankings/tags/browse/user 各根测试文件由 `Run(...)` 驱动。
  - 基线验证（全部 PASS）：`gopls check` 全仓无诊断；`go list ./...` exit 0；`go test ./...` 无 FAIL/panic；`go test -race ./...` 无 FAIL/DATA RACE；`go vet ./...` exit 0；`sh scripts/test-architecture.sh` 与 `sh scripts/test-documentation.sh` exit 0；`git diff --check` exit 0。无既有失败，无需归因。
- 验证证据：上面各命令的 exit code 与输出已记录；`rg --files -g '*.go' -0 | xargs -0 gopls check` 无诊断；`go list ./...` 70 包；全仓 test/race/vet、architecture/documentation 门禁、diff check 全绿。
- 剩余风险：无已知基线失败。goal-2 三文件处于 staged 状态（/goal 初始化所建），实施过程中保持不动；`.idea/` 不纳入提交。
- 下一步：Task 2。

## Task 2：建立 invocation、authstore 与 client 能力

- 状态：`completed`
- 范围：新增三个能力包，不迁移命令调用方，不删除 `app`。
- 必须完成：
  - 先为 `authstore.Open`、`client.New`、required/optional auth 行为建立聚焦测试。
  - 新增 `invocation.RootOptions`、`Streams` 与 `NewStreams`。
  - 从现有 `app` 复制并收敛配置/client、auth store 和认证流程实现，保持行为逐字一致。
  - 测试覆盖无账号、空 token、匿名、认证失败匿名重试、非认证错误、auto-relogin 的主要分支。
  - 保留 `app` 供旧命令暂时编译；本 task 不建立 app 到新包的兼容转发。
- 验证要求：三个新包的 test/race/vet、gopls、gofmt、全仓 `go test ./...`。
- 实际做了什么：
  - 新增 `internal/cli/invocation`：`RootOptions{Proxy,Host}`、`Streams{In,Out,Err}`、`NewStreams`；仅依赖 stdlib `io`，不导入 config/auth/SDK/Cobra/update。
  - 新增 `internal/cli/authstore`：`Open() (*auth.FileStore, *auth.Store, error)`，从 `app.OpenAuth` 原样迁移（AuthPath → EnsureDir → auth.Open），不吞错误。
  - 新增 `internal/cli/client`：`New(options, token)` 吸收 `app.LoadRuntime`+`NewClient`（私有 `resolveRuntime`+`buildClient` 拆分，避免 auth 流程重复解析）；`WithRequiredAuth(options, errOut, fn)` 从 `WithAuthedClient` 原样迁移并改名（`aio.Err` → `errOut`，`errOut==nil` 不写提示，重登只重试一次并持久化 token）；`WithOptionalAuth(options, errOut, fn)` 从 `WithOptionalAuthClient` 原样迁移（匿名/重试/非认证错误不重试语义保持）。
  - TDD：先写测试后实现。`authstore/store_test.go`（isolate HOME 后 Open 建目录+空 store、Commit 后重开持久化）；`client/client_test.go`（无网络构造、host 非法拒绝、device UUID 创建+落盘+复用、token 携带）；`client/auth_test.go`（optional 匿名/fallback 匿名 2 次+stderr 提示/非认证错误 1 次；required 无账户/空 token/无 auto-relogin/无密码/auto-relogin 经 httptest 重登并持久化 new-token）。
  - 保留 `app` 供旧命令编译；本 task 未建立 app→新包的兼容转发。
- 验证证据：`go test ./internal/cli/authstore/ ./internal/cli/client/` 全 PASS（verbose 逐条列出，含 httptest 重登路径）；`go test -race` 两包 ok；`go vet` 三包 exit 0；`gofmt -l` 清空；`gopls check` 新文件无诊断；全仓 `go test ./...` 无 FAIL/panic；`git diff --check` exit 0。
- 剩余风险：`client.New` 内部 `resolveRuntime` 会在缺失 device UUID 时创建 `~/.javdb-cli` 目录与 device_uuid 文件（与 app.LoadRuntime 一致）；测试均 isolate HOME。
- 下一步：Task 3。

## Task 3：迁移 app 调用方并删除 app

- 状态：`completed`
- 范围：迁移 root 与全部命令构造器、client/auth 调用和 update 组装，最终删除 `internal/cli/app`。
- 必须完成：
  - root 改用 `invocation.RootOptions` 与 `invocation.NewStreams`，保持 `New/Run` 签名和 AddCommand 顺序。
  - 逐命令替换 `app.Flags`、`app.IO`、`LoadRuntime/NewClient` 与两种 auth helper。
  - `commands/auth` 改用 `authstore.Open` 与 `client.New`。
  - 将 update proxy/coordinator/build info 迁入 `commands/update`，先补该包测试。
  - 将 version 根测试迁入 `commands/version`，锁定 build info 文本与 JSON。
  - 用 LSP/rg 确认 `internal/cli/app` 和 `app.*` 零引用后删除目录。
- 验证要求：所有命令包与 root 的 test/race/vet、gopls、全仓 test、build。
- 实际做了什么：
  - `internal/cli/root.go`：`app.Flags`/`app.NewIO` → `invocation.RootOptions`/`invocation.NewStreams`（局部变量 `flags`→`options`、`aio`→`streams`）；`New(stdin,stdout,stderr)` 与 `Run(args,stdin,stdout,stderr)` 签名、26 个 AddCommand 顺序、persistent `--proxy/--host`、Windows cleanup 不变。
  - 26 个命令包：`New(options *invocation.RootOptions, streams *invocation.Streams)`（config 仅 `New(streams)`，version 不变）；匿名命令 `app.LoadRuntime+app.NewClient(rt,"")` → `client.New(options,"")`；required auth `app.WithAuthedClient` → `client.WithRequiredAuth(options, streams.Err, fn)`；optional auth `app.WithOptionalAuthClient` → `client.WithOptionalAuth(options, streams.Err, fn)`；`app.OpenAuth` → `authstore.Open`（commands/auth）。
  - `commands/auth`：auth.go 组 + login/list/use/remove/check 全部改用 `client.New`+`authstore.Open`+`invocation`；login.go 保持"先建 client（config 错误先于提示）再交互提示"的原始顺序，`Short` 恢复 "interactive if flags omitted"（脚本误改后修正），本地 `client` 变量改名 `c` 避免与包名冲突；check.go 同。
  - `commands/update`：新增 `update_helpers.go`（未导出 `resolveProxy`/`newProductionCoordinator`/`buildInfo`，唯一消费者为 `update.New`）；`update.go` 改用它们与 `invocation.RootOptions/Streams`；新增 `update_test.go`（`--json` 无 `--check` 前置错误、help 文本、resolveProxy 配置优先级、newProductionCoordinator 离线构造）。
  - `commands/version`：新增 `version_test.go`（buildinfo 注入后文本 `javdb v0.1.0` + JSON `version=v0.1.0/commit=deadbeef`、help），接管根 `version_cmd_test.go`。
  - 删除 `internal/cli/app/`（app.go/auth.go/update.go/auth_test.go）；删除前 `rg` 确认全仓（含 sdk/scripts）零 `internal/cli/app` 导入、零 `app.*` 符号引用。
  - 架构门禁过渡同步：required_dirs 移除 `internal/cli/app`、新增 `invocation/client/authstore`，forbidden 增加 `internal/cli/app`。
  - 迁移过程中用脚本处理 60 个文件，手动修正脚本缺陷：LoadRuntime/NewClient 配对、WithAuthed/OptionalAuth 替换顺序、`flags`→`options` 误改字符串（login Short）与注释（root.go 注释恢复 "persistent flags"）、entity.go 误加 `client` import（局部参数遮蔽，移除）、auth/login+check 局部 `client` 变量改名。
- 验证证据：`go build ./...` exit 0；全仓 `go test ./...` 与 `go test -race ./...` 无 FAIL/panic/DATA RACE；`go vet ./...` exit 0；`gopls check` 全仓无诊断；`gofmt -l` 清空；`git diff --check` exit 0；`sh scripts/test-architecture.sh`（过渡态）与 `sh scripts/test-documentation.sh` exit 0；`sh scripts/build.sh` 产出 build/javdb（commit=9e8f965）；`build/javdb --help` exit 0、`version --json` 正常；根 help 全量字面契约 `TestRootHelpFullLiteral` PASS；26 个命令 `New` 签名已全部为 invocation 类型。
- 剩余风险：命令测试文件（commands/*/*_test.go）仍随命令包编译，其中部分 `invocation` 导入由脚本补入；`commands/auth` 的 stderr 提示（重新登录/匿名）文本在 `client` 包中保持逐字。`movie/magnet/entity` 尚未迁移（Task 4）。
- 下一步：集中检查-debug 1。

## 集中检查-debug 1：能力拆分

- 状态：`completed`
- 检查范围：Task 1–3 的行为偏离、依赖方向、重复实现、认证安全、LSP、测试与工作树。
- 必须完成：
  - 确认不存在 app facade、service locator、循环依赖或 command-to-command import。
  - 对比认证与 update 错误文本、stderr 提示、校验顺序和回调次数。
  - 确认 root help、命令集合、persistent flags 与 version 输出契约未变。
  - 发现问题时在 Task 4 前追加修复 task。
- 验证要求：聚焦 race、全仓 test/vet/build、架构门禁当前状态、gopls、diff check。
- 实际做了什么：
  - 结构：`go list ./...` 无 import cycle；`internal/cli` 下无 app/session/output/new 等 service-locator 包；`go list -f` 遍历 26 个命令包确认无 command-to-command import。
  - 依赖方向：client 只依赖 invocation+authstore+config paths/settings+sdk；authstore 只依赖 config paths+storage/auth；invocation 只依赖 stdlib io；均不依赖 CLI 命令、不构成循环。
  - 认证契约：`client.WithRequiredAuth` 的 4 条错误文本（no default account / empty token / expired without auto-relogin / no saved password）与旧 `app.WithAuthedClient` 逐字一致；stderr 提示 `缓存 token 已失效，重新登录…` 与 `token 无效，改用匿名请求…` 逐字一致；回调次数（匿名 1 次、fallback 2 次、非认证错误 1 次、auto-relogin 2 次）由 client/auth_test.go 断言。
  - update 契约：`--json` 必须配合 `--check` 的前置校验在 resolveProxy/newProductionCoordinator 之前（update.go RunE 顺序）；update_test.go 断言错误文本。
  - CLI 契约：`TestRootHelpFullLiteral`、`TestKeySubcommandHelpLiterals`、`TestPersistentFlagsLocked`、`TestNoNetworkParameterErrorsExact`、`TestRootCommandSetMatchesHelp` 全 PASS；version 文本/JSON 由 commands/version/version_test.go 锁定。
  - LSP/测试/工作树：全仓 gopls 无诊断；`go test ./...` 62 包全 ok；`go test -race ./internal/cli/...` 无 FAIL/DATA RACE；`go vet ./...`、`gofmt -l`、`git diff --check` 全 clean；`goal-2` 三文件与 `.idea/` 未动。
- 验证证据：上面各项 exit code 与输出；结构/依赖/文本检查结果已列。
- 剩余风险：无需要追加修复 task 的问题。movie/magnet/entity 仍待 Task 4 迁移到 result。
- 下一步：Task 4。

## Task 4：建立 result 并收窄 entity

- 状态：`completed`
- 范围：统一 movie/magnet/named 的纯投影，迁移全部调用方，删除旧 movie/magnet 包。
- 必须完成：
  - 先迁移原 `movie_test.go`、`magnet_test.go` 和 entity 的 named 投影测试到 `result`。
  - 按 plan 固定的类型和函数名实现 `MovieRow`、`MagnetRow`、`NamedRow`。
  - 保持所有字段 fallback、数值转换、flag 顺序、日期、大小、Line/HashLine 与过滤语义。
  - 将 entity 的 named 投影移出，`Execute` 改用 `FilterMoviesWithMagnets`。
  - 逐命令迁移 movie/magnet/named 调用点，不统一 writer 或 JSON helper。
  - 用 LSP/rg 确认旧包与旧符号零引用后删除 `movie/`、`magnet/`。
- 验证要求：result/entity 聚焦 test/race/vet、全部调用命令测试、gopls、全仓 test/build。
- 实际做了什么：
  - 新增 `internal/cli/result`（movie.go/magnet.go/named.go）：`MovieRow`+`Line`+`ProjectMovie`+`ProjectMovies`+`FilterMoviesWithMagnets`；`MagnetRow`+`Line`+`HashLine`+`ProjectMagnet`+`ProjectMagnets`；`NamedRow`+`Line`+`ProjectNamed`+`ProjectNamedAll`。原 `magnet.Flags/FormatSize` 改为未导出 `flags/formatSize`（+`joinComma`/`truthy`），共享 `display/intValue` 留在 movie.go 供全包使用。语义与旧 movie/magnet 逐字一致。
  - TDD：先建 `result/{movie,magnet,named}_test.go`（迁移原 movie_test/magnet_test 全量 + entity 的 4 个 named 投影测试，并补充 Line/HashLine 字面断言），跑绿后再迁移调用方。
  - 调用方迁移（脚本 + 手工修正）：8 个命令 `movie.FilterHasMagnets` → `result.FilterMoviesWithMagnets`；`movie.ProjectAll` → `result.ProjectMovies`（11 文件）；`movie.Project` → `result.ProjectMovie`（top250）；`magnet.ProjectAll` → `result.ProjectMagnets`（magnets_lines/detail_lines）；`entity.ProjectNamedAll` → `result.ProjectNamedAll`（search/collections/rankings.actors）。
  - entity 收窄：删除 `NamedRow`/`(NamedRow) Line()`/`ProjectNamed`/`ProjectNamedAll` 与 scalar import（仅 named 投影使用）；`Execute` 改用 `result.FilterMoviesWithMagnets`；包 doc 更新为"只保留六实体查询用例，命名投影位于 result"。
  - entity_test.go 删除 4 个 named 投影测试（已迁 result/named_test.go），保留全部 Execute 用例；删除脚本误加的 `result` import（`result` 是测试局部变量）。
  - 删除 `internal/cli/movie/`、`internal/cli/magnet/`；删除前 `rg` 确认零外部导入、零旧符号引用（entity.NamedRow/ProjectNamed/ProjectNamedAll 外部零引用）。
  - 架构门禁过渡同步：required_dirs 移除 movie/magnet、新增 result；forbidden 增加 movie/magnet；新增 result 依赖规则（只允许 stdlib+scalar）与 invocation 依赖规则（只允许 stdlib）。
- 验证证据：`go test ./internal/cli/result/` 全 PASS（TDD 先行）；`go build ./...` exit 0；全仓 `go test ./...`、`go test -race ./...`、`go vet ./...`、gopls、gofmt、git diff 全绿；`sh scripts/test-architecture.sh` 与 `test-documentation.sh` exit 0；`sh scripts/build.sh` + `build/javdb --help`/`version --json` 冒烟正常；`go list ./internal/cli/result` 依赖仅 fmt/scalar/strconv。
- 剩余风险：无。命令内 `writeMovieRows`/`writeJSON` 写循环仍由各命令 owner 持有（plan 允许）。
- 下一步：Task 5。

## Task 5：把命令专属根测试迁回 owner

- 状态：`completed`
- 范围：处理根目录 12 个命令专属测试文件，保留或加强行为覆盖但消除重复。
- 必须完成：
  - 按 `plan.md` 的映射逐文件核对根断言与 owner 现有断言。
  - 相同断言删除重复；owner 缺少的 help、flag、前置参数或输出断言迁入 owner 测试。
  - 六实体命令各自锁定自己的命令名、Use、共同 flags 与必要默认值。
  - lists/rankings 的所有子命令由对应命令组测试覆盖。
  - update/version 拥有自己的测试文件。
  - 删除弱 `AuthRequired.Error()` 断言，确认 client auth 控制流已有真实覆盖。
- 验证要求：所有受影响命令包 test/race/vet、gopls、全仓 test。
- 实际做了什么：
  - 逐文件核对根测试与 owner 现有断言（goal-1 已为每个命令建 owner 测试）：
    - download 根 3 个测试（help 文档/必须选输出/空白输出拒绝）与 `commands/download/download_test.go` 完全重复 → 删根。
    - entity 根 TestEntityHelp（6 命令共同 flags）与 6 个实体 owner `TestNewBuilds*Command`（zone/tag/main/sort/order/page/limit/all/has-magnets/json + 默认值断言）重复 → 删根。
    - magnets 根 TestMagnetsHelp 与 owner（含 "not requires login"）重复 → 删根。
    - search 根 TestSearchHelpListsFlags 与 owner `TestNewHelpListsExpectedFlags` 重复 → 删根。
    - tags/browse 根 TestTagsBrowseHelp 与两个 owner 重复 → 删根。
    - rankings/top250 根 TestRankingsTop250Help 与 rankings owner（组+movies/actors/playback flags）+ top250 owner（help）覆盖 → 删根。
    - lists 根 TestListsHelp 与 lists owner（组+show 子命令）覆盖 → 删根。
    - user 根 TestUserCmdsHelp 与 watched/want/recent/collections/mark/unmark 6 个 owner 覆盖 → 删根。
    - update/version 根测试已在 Task 3 迁移到对应 owner。
  - 根测试独有的断言迁移：comments owner `TestNewHelpListsFlags` 增补 "never fetches later pages"（Long 文本）；detail owner `TestNewHelpListsFlags` 增补 "must not require login"（无 "needs login"）。
  - 删除弱断言 `TestAuthRequiredMessageWithoutAutoRelogin`（只检查 `AuthRequired.Error() != ""`）；client `WithRequiredAuth` 的真实控制流（无账户/空 token/无 auto-relogin/无密码/重登持久化）由 `client/auth_test.go` 覆盖。
  - 删除根目录 12 个命令专属测试文件：comments_cmd_test/detail_test/download_cmd_test/entity_cmd_test/lists_cmd_test/magnets_cmd_test/rankings_cmd_test/search_cmd_test/tags_browse_cmd_test/update_cmd_test/user_cmd_test/version_cmd_test。
- 验证证据：`go test ./...` 无 FAIL/panic；`go test -race ./internal/cli/...` 无 FAIL/DATA RACE；`go vet ./internal/cli/...`、gopls、gofmt、git diff 全绿；根目录只剩 contract_test.go。
- 剩余风险：无。所有被删根断言的等价或更强断言已在 owner 存在。
- 下一步：Task 6。

## Task 6：收敛唯一 root_test.go

- 状态：`completed`
- 范围：将 CLI-wide 契约集中到一个 `root_test.go`，清空其他根测试文件。
- 必须完成：
  - 将 `contract_test.go` 重命名/整理为 `root_test.go`。
  - 保留根 help 全量字面、命令集合、关键命令组最终 help、persistent flags 和全局错误/退出码。
  - 删除已由命令 owner 覆盖的单命令 flags 与局部校验。
  - 检查根目录除 `root.go/root_test.go` 外无其他 `.go` 文件。
  - 对比重整前后的关键 `Run(... --help)` 和无网络错误输出字节。
- 验证要求：root test/race/vet、构建产物离线 help/version 冒烟、gopls、全仓 test。
- 实际做了什么：
  - `internal/cli/contract_test.go` 重命名为 `internal/cli/root_test.go`，更新包 doc 注释为"根目录唯一测试文件，锁定 CLI 稳定对外行为；命令专属 flag/前置校验由 owner 拥有"。
  - 根目录现在只有 `root.go` + `root_test.go`（`ls internal/cli/*.go` 确认）。
  - `root_test.go` 保留：TestRootHelpFullLiteral（根 help 全量字面）、TestKeySubcommandHelpLiterals（9 个关键命令组最终树完整 help 字面）、TestPersistentFlagsLocked（--proxy/--host 默认值+usage）、TestNoNetworkParameterErrorsExact（缺参/未知命令/update --json 的 stderr+exit 字面）、TestRootCommandSetMatchesHelp（命令集合）。
  - 字节对比：`build/javdb frobnicate` → `unknown command "frobnicate" for "javdb"` exit 1；`build/javdb search` → `accepts 1 arg(s), received 0` exit 1；`build/javdb update --json` → `--json is only supported with --check` exit 1；根 help 字面契约 PASS。
- 验证证据：`go test ./internal/cli/ -v` 5 个根契约测试全 PASS；全仓 test/race/vet/gopls/gofmt/diff、architecture/documentation 门禁全绿；构建产物 help/version 冒烟正常；`ls internal/cli/*.go` 仅 root.go/root_test.go。
- 剩余风险：无。
- 下一步：集中检查-debug 2。

## 集中检查-debug 2：结果层与测试 owner

- 状态：`completed`
- 检查范围：Task 4–6 的纯包边界、输出契约、测试缺口、重复和死代码。
- 必须完成：
  - 确认 result 无 IO/Cobra/SDK/config/storage/update 依赖。
  - 确认 entity 只含查询用例，不再含 NamedRow 投影。
  - 确认 root 没有命令专属测试，所有迁移断言有明确 owner。
  - 对比 movie/magnet/named 的文本和过滤契约，检查数值边界与缺失字段。
  - 发现问题时在 Task 7 前追加修复 task。
- 验证要求：result/entity/root/commands race、全仓 test/vet/build、gopls、diff check。
- 实际做了什么：
  - result 边界：`go list` 依赖仅 fmt/scalar/strconv（无 io/Cobra/SDK/config/storage/update），架构门禁已强制；未接收 writer、无空列表文案、无 JSON 编码。
  - entity 收窄：导出仅 `Options`/`Result`/`Execute`，NamedRow/ProjectNamed/ProjectNamedAll 已移出；`Execute` 用 `result.FilterMoviesWithMagnets`。
  - root：目录只有 `root.go`/`root_test.go`；12 个命令专属测试已删除，owner 覆盖等价或更强断言。
  - 输出契约：`go test ./internal/cli/result/ -v` 15 个测试全 PASS（movie 字段/缺 release date/float ID 截断/顺序、FilterMoviesWithMagnets 缺失字段保留+数值变体、magnet name/title 降级/日期截断/GB-MB/flag 顺序/Line/HashLine、named 中文优先/count fallback/无 count/Line）。
  - 死代码/重复：`rg` 确认 movie/magnet/app 路径与 import 全消失；命令包间无 command-to-command import；entity 无残留 named 投影。
  - LSP/测试：全仓 gopls 无诊断；`go test ./...`、`go test -race ./...`、`go vet ./...`、gofmt、git diff 全绿；architecture/documentation 门禁 exit 0。
- 验证证据：上面各项输出已列；result 15 测试逐条 PASS；包图/依赖/路径检查零命中。
- 剩余风险：无需要追加修复 task 的问题。
- 下一步：Task 7。

## Task 7：同步维护者文档与架构门禁

- 状态：`completed`
- 范围：让维护者文档和 `scripts/test-architecture.sh` 描述并强制最终结构。
- 必须完成：
  - 更新 architecture 总体流程、包职责和目录约定。
  - 更新 development 目录地图与依赖方向。
  - 更新 ADR 0001 中 CLI 内部边界的最终决策，保留历史决策语境。
  - 门禁 required dirs 改为 invocation/client/authstore/result/entity，禁止 app/movie/magnet。
  - 门禁强制 CLI 根仅有 `root.go/root_test.go`，并增加 invocation/result 依赖规则。
  - 不修改 goal-1、public docs、README、skill 或 changelog。
- 验证要求：shell syntax、architecture/documentation 门禁、链接/路径检索、`git diff --check`。
- 实际做了什么：
  - `docs/maintainers/architecture.md`：总体流程与 ASCII 图改为 `cli/{invocation,client,authstore,result,entity}`；包职责删除 `cli/app`/`cli/movie`/`cli/magnet` 条目，新增 invocation/client/authstore/result/entity 各职责，`commands/update` 注明拥有 proxy/coordinator/buildinfo。
  - `docs/maintainers/development.md`：目录地图移除 app/movie/magnet，新增 invocation/client/authstore/result/entity + `root_test.go`；依赖方向段落同步。
  - `docs/maintainers/adr/0001`：决策与依赖约束改为 `cli/{invocation,client,authstore,result,entity}`，说明 `internal/cli` 根只保留 root.go/root_test.go，result 只依赖 stdlib+scalar、invocation 只依赖 stdlib，不建立 service-locator/output 大包。
  - 架构门禁：required_dirs 已含 invocation/client/authstore/result/entity 且禁止 app/movie/magnet（Task 2/4 已加）；新增"CLI 根目录除 root.go/root_test.go 外无其他 Go 文件"与"root_test.go 必须存在"规则；result/invocation 依赖规则已在 Task 4 加。
  - 未修改 goal-1、public docs、README、skill、changelog；`rg` 确认 maintainer docs 无 cli/app|cli/movie|cli/magnet|commands/{account,catalog,user}|shared/values 残留。
- 验证证据：`sh -n scripts/test-architecture.sh` + `sh scripts/test-architecture.sh` exit 0；`sh scripts/test-documentation.sh` exit 0；其余四个离线门禁 exit 0；全仓 test/race/vet/gopls/gofmt/git diff 全绿；`sh scripts/build.sh` + `build/javdb --help`/`version --json` 冒烟正常。
- 剩余风险：无。
- 下一步：Task 8。

## Task 8：最终全量验证与交付审计

- 状态：`completed`
- 范围：核对 input、plan、最终代码、测试、文档、安全和用户工作树。
- 必须完成：
  - 确认 `app/movie/magnet` 路径和 import 零残留，无兼容 wrapper 或死符号。
  - 确认目标包图无循环、无 command-to-command 依赖、CLI 只通过 sdk 远程访问 JavDB。
  - 确认根目录只有 `root.go/root_test.go`，所有新能力有聚焦测试。
  - 运行全仓 LSP、test、race、vet、build、架构与文档门禁、gofmt/diff check。
  - 用构建产物离线验证根 help、关键命令 help 和 `version --json`。
  - 检查 `.idea/` 与其他用户文件未被修改，列出最终 diff manifest。
  - 记录未运行的真实凭据/API/在线 update，并说明剩余风险。
- 验证要求：`plan.md` 第 13 节全部命令有成功证据；任何失败必须显露真因，不得伪装完成。
- 实际做了什么：
  - 结构审计：`rg` 全仓（internal/sdk/cmd/scripts）`internal/cli/(app|movie|magnet)"` import 与 `app.*`/`movie.*`/`magnet.*`/`entity.NamedRow|ProjectNamed*` 符号零命中；`internal/cli/{app,movie,magnet}` 目录不存在；`internal/cli` 根目录只有 `root.go`/`root_test.go`；命令包间无 command-to-command import；`go list ./...` 71 包无循环。
  - 能力测试：authstore 2 测试、client 8 测试（含 httptest 重登持久化）、result 15 测试、entity 6 测试（Execute 全路径）、invocation 无包；26 个命令包 owner 测试、update 4 测试、version 2 测试、root 5 个契约测试。
  - 全量验证（plan §13 全部命令）：`gopls check` 全仓无诊断；`go test ./...` 无 FAIL；`go test -race ./...` 无 FAIL/DATA RACE；`go vet ./...` exit 0；`sh scripts/build.sh` 产出 build/javdb（commit=9e8f965）；`gofmt -l` 清空；`git diff --check` clean。
  - 离线冒烟：`build/javdb --help`、`version --json`、`config --help`、`top250 --help`、`rankings movies --help` 全部 exit 0；根 help 全量字面 `TestRootHelpFullLiteral` PASS；无网络错误输出（未知命令/缺参/update --json）exit 1 + 文本正确。
  - 门禁：architecture/documentation/releasenotes/package-release/homebrew-formula/workflows 六个离线脚本全部 exit 0；`go test ./scripts/login_probe.go` 编译通过。
  - 工作树：`.idea/` 未纳入提交；README 双语、changelog、docs/en+zh-CN、skills、sdk 目录零改动；goal-2 三文件 staged+tasks.md 已回写；`git status` 101 项全部为 goal-2 代码/文档改动 + goal-2 文件 + `.idea/`。
  - 未运行真实凭据、真实 JavDB API、在线 update、发布或任何 `--apply` 外部写入。
- 验证证据：上面各项 exit code 与计数；结构/符号/import 扫描零命中；SDK contract 12 PASS/0 FAIL（`go test ./sdk/`）；根契约 5 PASS。
- 剩余风险：无已知高风险问题。本 goal 未 commit（HEAD 保持 9e8f965）；提交/推送/发布等待用户授权。
- 下一步：无；goal-2 可标记完成。
