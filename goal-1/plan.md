# Goal 1 计划：移除内部兼容层并按真实命令重整 Go 目录

## 1. 目标与基线

本目标是上一阶段领域拆分后的结构收敛，不新增产品能力。实施必须以开始 Task 1 时记录的当前 dirty worktree 为唯一行为基线，并保持工作树中所有用户改动。

计划编写时观测到的引用为：

```text
HEAD        3da5d6f85051c951b29b6fdc084bb49e1780c140
origin/main e84ad613d9867759b2aaeb3e50f6d02a933b2887
历史参考     9c3ee65b3b5d301461cf7ace7b265fdfe202cfa2
```

Task 1 必须重新记录实际值；若届时引用变化，以当时工作树为准并回写任务证据，不通过 reset 或 checkout 恢复上述值。

目标结果：

- CLI 只按真实命令组织，不保留 `catalog`、`user`、`account` 等非命令目录。
- 删除 CLI、config、update、release-note 的内部兼容 facade/compat。
- App API 根包变为真实 Client 组合层；协议能力继续位于 endpoint 子包。
- 建立纯 JSON、标量、影片、磁力和实体投影边界，不建立横切 output/printer/render。
- 保持公开 SDK、CLI、配置、认证、更新和 release-note 行为。

## 2. 不可违背的契约

### 2.1 公开 SDK

- import path 仍为 `github.com/FlanChanXwO/javdb-cli/sdk`，package name 仍为 `javdb`。
- 导出常量、类型、函数、方法签名及 `errors.Is`/`errors.As` 行为不变。
- `Client.API() *appapi.Client` 保留，返回对象原有扁平方法集继续可编译、可调用。
- `RefreshTagTaxonomy` 与 `LoadOrRefreshTaxonomy` 的返回类型不变。
- 不增加新的 internal 类型泄漏；现有两个冻结例外不扩大。

### 2.2 CLI

- 命令和子命令名称、注册顺序、参数、flag、默认值、help 文本不变。
- stdout、stderr、文本列、空结果文案、JSON 字节、尾随换行和退出码不变。
- 命令执行远程 JavDB 操作时只通过公开 SDK，不直接导入 App API 或协议包。
- 本机 config、storage 和显式 update 不属于远程 JavDB 操作，可直接依赖其真实内部职责包。

### 2.3 本机状态与工具

- 配置优先级保持 flag > environment > file > defaults。
- config、auth、device UUID、tag taxonomy 的路径、格式和权限不变。
- update 来源识别、SemVer、checksum、archive、候选校验和替换行为不变。
- release-note 子命令、flag、dry-run/apply 边界、文本和 JSON 不变。
- 不输出密码、JWT、`auth.json` 内容或其他 secret。

## 3. 允许破坏的内部契约

以下内部入口不再保留兼容性：

- 根 CLI 的渲染、输入和测试 wrapper。
- `internal/config` 根 facade。
- `internal/update` 的 alias、forwarder 和根 alias types。
- App API 根包的类型/helper forwarder；只保留公开 SDK 所需的真实 `Client` 组合。
- `scripts/releasenotes/compat.go`。
- 旧 internal import path 以及只验证转发身份的 facade tests。

任何删除都必须先通过 gopls references、Go 包图和文本检索确认调用方已经迁移，并由对应测试证明行为仍受覆盖。

## 4. 纯共享包

### 4.1 `internal/common/jsonx`

```go
func ObjectArray(raw json.RawMessage) []map[string]any
func ObjectSlice(value any) []map[string]any
func RawString(raw json.RawMessage) string
func MarshalLine(value any) ([]byte, error)
```

- `ObjectArray`、`ObjectSlice` 保持当前 nil、非法 JSON、null 元素和非 object 元素语义。
- `RawString` 保持现有原始标量字符串语义，不擅自 unescape 或规范化。
- `MarshalLine` 使用 `json.Encoder`、`SetEscapeHTML(false)`，成功时包含且只包含一个尾随换行。
- 包内不得接受 `io.Writer`、写输出、包含 CLI 文案或吞掉编码错误。

### 4.2 `internal/common/scalar`

```go
func String(value any) string
func Int64(value any) (int64, bool)
```

保持当前数值类型、浮点截断、溢出和非法输入语义。CLI 的浮点 ID 展示等领域差异不得并入 scalar。

`internal/common` 根目录不建立 Go package。

## 5. CLI 领域投影

### 5.1 `internal/cli/movie`

```go
type Row struct {
	Number      string
	ID          string
	Title       string
	ReleaseDate string
}

func Project(item map[string]any) Row
func ProjectAll(items []map[string]any) []Row
func FilterHasMagnets(items []map[string]any) []map[string]any
```

只处理影片记录的字段投影与 `has-magnets` 过滤。不得包含 IO、Cobra、SDK 调用、JSON 编码、排行、磁力、详情或空列表文案。

### 5.2 `internal/cli/magnet`

```go
type Row struct {
	Name      string
	Size      string
	Flags     []string
	CreatedAt string
	Hash      string
}

func Project(item map[string]any) Row
func ProjectAll(items []map[string]any) []Row
func FormatSize(value any) string
```

- `detail --magnets` 与 `magnets` 共同使用该投影。
- truthy、名称降级、日期、大小和 hash 语义保持当前行为。
- 包只返回结构化 Row；命令负责写出既有文本。
- 包不得依赖 IO、Cobra、SDK client 或 JSON writer。

### 5.3 `internal/cli/entity`

```go
type Options struct {
	Zone       string
	Sort       string
	Order      string
	Page       int
	Limit      int
	TagRefs    []string
	Main       []string
	AllPages   bool
	HasMagnets bool
}

type Result struct {
	Entity   map[string]any
	EntityID string
	Movies   []map[string]any
}

type NamedRow struct {
	ID       string
	Name     string
	Count    any
	HasCount bool
}

func Execute(ctx context.Context, client *javdb.Client, kind, ref string, options Options) (Result, error)
func ProjectNamed(item map[string]any) NamedRow
func ProjectNamedAll(items []map[string]any) []NamedRow
```

- `Execute` 统一实体解析、tag 解析、单页/全部页查询、影片过滤和实体 metadata 行为。
- actor、series、maker、director、code、list 各自拥有真实 Cobra command、`Use`、`Short`、`Args` 和 flag 绑定。
- entity 不创建 Cobra command、不写输出、不编码 JSON。

## 6. App API 组合

`internal/javdb/appapi/endpoint/*` 从接收 transport 的包级函数改为有状态 capability：

```text
auth.AuthEndpoint
browse.BrowseEndpoint
entity.EntityEndpoint
lists.ListsEndpoint
movie.MovieEndpoint
rankings.RankingsEndpoint
search.SearchEndpoint
user.UserEndpoint
media.MediaEndpoint
```

`endpoint/magnets` 保持纯 helper，不制造无状态 service。

构造顺序固定为：

```text
transport
  -> auth, browse, search, lists, rankings
  -> movie(search)
  -> entity(browse, search)
  -> user(movie)
  -> media(fetcher)
```

根 `appapi.Client`：

- 只构造一次 transport。
- 通过根包内未导出的类型别名嵌入 transport 与各 capability。
- 使用 method promotion 保留 `Client.API()` 调用方当前可见的方法集。
- 不暴露可访问的 endpoint 字段。
- 不保留手写一行式 capability forwarder。

endpoint 不得导入 appapi 根包、SDK 或 CLI。SDK 可直接引用 model 以及排行/磁力纯 helper，但对外类型名与签名必须保持不变。

`scripts/login_probe.go` 改用公开 SDK 创建 client，再通过 `API()` 调用 Startup、Users、ResolveUserID；默认验证只编译该脚本，不运行真实登录。

## 7. Config 与 Update

### 7.1 Config

- `internal/config` 根目录不建立 Go package。
- 路径、目录和文件定位全部由 `config/paths` 提供。
- schema、默认值、环境变量、优先级、读写和校验全部由 `config/settings` 提供。
- SDK、CLI app、config 命令和 tags storage 直接依赖正确子包。
- 根配置测试迁至 settings 或真实调用方。

### 7.2 Update

根包只保留：

```text
internal/update/coordinator.go
internal/update/interfaces.go
internal/update/coordinator_test.go
```

- Coordinator 在 `interfaces.go` 定义自己需要的最小依赖接口，不 alias 子包接口。
- `Execute` 直接使用 `update/model.Request` 与 `update/model.Result`。
- CLI app 从 release/source/process/archive 直接组装生产依赖。
- CLI root 直接调用 process 的 Windows pending cleanup。
- update 命令和状态输出直接使用 model 类型。
- 删除 facade、facade test 和根 alias types。

## 8. CLI 命令布局

最终命令目录为：

```text
internal/cli/commands/
├── auth/{auth.go,check.go,list.go,login.go,prompt.go,remove.go,use.go}
├── config/config.go
├── search/search.go
├── detail/{detail.go,detail_lines.go}
├── comments/comments.go
├── magnets/magnets.go
├── download/download.go
├── tags/tags.go
├── browse/browse.go
├── actor/actor.go
├── series/series.go
├── maker/maker.go
├── director/director.go
├── code/code.go
├── list/list.go
├── watched/watched.go
├── want/want.go
├── recent/recent.go
├── collections/collections.go
├── mark/mark.go
├── unmark/unmark.go
├── rankings/{rankings.go,movies.go,actors.go,playback.go}
├── top250/top250.go
├── lists/{lists.go,show.go,search.go,related.go}
├── update/{update.go,status.go}
└── version/version.go
```

规则：

- 每个目录必须对应真实命令或真实命令组。
- 每个目录必须有与目录同名的主文件。
- 禁止以 `command.go`、`output.go`、`printer.go`、`render.go` 作为泛化主文件。
- auth 拥有 prompt 及 login/list/use/remove/check。
- rankings 拥有 movies/actors/playback；top250 保持独立顶层命令。
- lists 拥有 show/search/related。
- 每个命令持有自己的 Cobra metadata、参数校验、flag、文本和 JSON 写入。
- JSON 命令调用 `jsonx.MarshalLine` 后写入 command writer，并传播编码与写入错误。
- detail 与 magnets 使用 magnet；影片列表使用 movie；命名实体列表使用 entity。
- `internal/cli/root.go` 直接组装命令树并实现 Run。

最终删除：

```text
internal/cli/facade.go
internal/cli/root/
internal/cli/input/
internal/cli/output/
internal/cli/printer/
internal/cli/render/
internal/cli/commands/account/
internal/cli/commands/catalog/
internal/cli/commands/user/
```

## 9. Release-note 工具

- `scripts/releasenotes/main.go` 只负责参数解析和子命令分派。
- 删除 `scripts/releasenotes/compat.go`。
- validate/render 测试迁至 document。
- GitHub HTTP、PR 和 contributor 测试迁至 github/audit。
- prepare、coverage 和版本 bump 测试迁至 prepare/document。
- sync-history 测试迁至 history。
- root main test 只保留分派、help 和未知命令错误。
- 不改变 GitHub API、JSON、文本或 apply 行为。

## 10. 架构门禁

`scripts/test-architecture.sh` 必须在每个结构任务中同步调整相关规则，最终固定：

- 精确的命令目录 allowlist。
- 命令目录同名主文件存在。
- 旧路径和泛化主文件不存在。
- `go list ./...` 可加载且无 import cycle。
- `cmd/javdb` 只委托 `internal/cli`。
- CLI 命令不导入 App API 或协议包。
- endpoint 不反向导入 appapi 根包、SDK 或 CLI。
- common 不依赖 CLI、SDK、App API、config 或 update。
- release-note 实现不依赖 CLI 或 SDK。
- SDK internal imports 只允许最终设计明确需要的 appapi Client、model、纯 helper、config settings 和 taxonomy 类型。
- `go doc -all` 不新增公开 internal 类型泄漏。

## 11. 实施顺序

详细轮次与验收证据见 `tasks.md`。固定顺序为：

1. 冻结基线与建立公开契约测试。
2. 迁移 common。
3. 重构 App API 与 SDK 内部调用。
4. 删除 config/update facade。
5. 建立 CLI 共享领域与基础命令。
6. 按影片、实体、个人状态/列表/排行分批迁移真实命令。
7. 切换最终 CLI root 并清理旧路径。
8. 清理 release-note compat。
9. 更新维护者文档和最终架构门禁。
10. 完整离线回归与终审。

每三个普通 task 后执行一次集中检查，不跨 task 顺手实施相邻工作。

## 12. 验证矩阵

每个普通 task 至少运行：

```bash
gopls check <本 task 相关 Go 文件>
go test <相关包>
go test -race <相关包>
go vet <相关包>
gofmt -l <相关 Go 文件>
git diff --check
```

每个集中检查至少运行：

```bash
go list ./...
go test ./...
go vet ./...
sh scripts/test-architecture.sh
sh scripts/test-documentation.sh
git diff --check
```

最终回归：

```bash
rg --files -g '*.go' -0 | xargs -0 gopls check
go test ./...
go test -race ./...
go vet ./...
sh scripts/build.sh
build/javdb --help
build/javdb version --json
sh scripts/test-architecture.sh
sh scripts/test-documentation.sh
sh scripts/test-releasenotes.sh
sh scripts/test-package-release.sh
sh scripts/test-homebrew-formula.sh
sh scripts/test-workflows.sh
go test ./scripts/login_probe.go
gofmt -l $(rg --files -g '*.go')
git diff --check
```

不得运行真实 API、真实凭据、在线更新、发布或任何 `--apply` 外部写入路径。

## 13. 验收标准

- Go 包图不再包含旧 CLI 分组、CLI facade/input/output/root、config 根包、shared/values 或 release-note compat。
- App API 根 Client 没有手写 capability forwarder，且 `Client.API()` 原方法集仍可编译和调用。
- command 目录严格对应真实命令，主文件与目录同名。
- CLI command 包不导入 App API 或协议；远程操作只经 SDK。
- common、movie、magnet 不执行 IO、不创建 Cobra command、不调用 SDK。
- entity 只执行实体用例和纯投影，不创建 Cobra command、不写输出。
- SDK public API、CLI help、flag、文本、JSON、stderr 和退出码契约测试全部通过。
- gopls、test、race、vet、build 和全部离线脚本通过。
- 当前用户文件与未提交改动完整保留。
- 未新增 timeout、retry、截断、静默 fallback 或无证据限制。
- 未创建 commit、分支、push 或 release。

## 14. 失败与回滚边界

- 不使用 `git reset`、宽范围 checkout 或 stash 覆盖 dirty worktree。
- task 未通过聚焦测试或 LSP 时保持 `pending`，记录真实失败并只修复当前 task 范围。
- 删除旧路径前先完成调用方迁移、引用检查和替代测试；不得依靠失败后恢复文件来保证安全。
- 与用户已有改动冲突时保留用户内容，若无法无损继续则将 task 标记阻塞并记录精确冲突。
- 计划、实施和验证均不自动提交；提交、推送和发布等待用户另行明确授权。
