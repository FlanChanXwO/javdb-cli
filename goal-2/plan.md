# Goal 2 实施计划

## 1. 总览

本目标只重整 `internal/cli` 的内部包边界和测试 owner。最终运行流保持为：

```text
cmd/javdb
  -> internal/cli.New / Run
  -> internal/cli/commands/*
       -> internal/cli/client -> sdk
       -> internal/cli/authstore -> internal/storage/auth
       -> internal/cli/result   （纯投影与过滤）
       -> internal/cli/entity   （六实体查询用例）
       -> internal/update/*     （仅 commands/update）
```

不建立根依赖容器、Session、通用 writer 或兼容 facade。命令构造器直接接收所需的 root options 与 streams，命令仍是用户输入、Cobra 和输出语义的 owner。

## 2. 不可改变的契约

### 2.1 CLI

- `New(stdin io.Reader, stdout, stderr io.Writer) *cobra.Command` 与 `Run(args []string, stdin io.Reader, stdout, stderr io.Writer) int` 不变。
- 26 个真实顶层命令、各子命令、Cobra 注册结果、completion/help、persistent flags 与 help 字面不变。
- `--host`、`--proxy` 的默认值、usage 与配置优先级不变。
- 参数错误必须继续在远程请求前发生；stderr 文本、stdout 空值和退出码保持。
- 文本列、磁力 URI、空列表文案、JSON HTML escaping 与单尾随换行保持。
- required auth、optional auth、自动重登、匿名重试、token 持久化与提示文案保持。
- update 仍只由显式 `javdb update` 触发，来源识别、检查、安装与 JSON 限制不变。

### 2.2 SDK 与内部依赖

- CLI 的远程 JavDB 操作仍只能通过顶层 `sdk/` public facade。
- 不修改 `sdk` 导出符号、方法签名、错误类型或 `errors.Is`/`errors.As` 行为。
- 不让 `internal/javdb/appapi` 或 `internal/javdb/protocol/*` 泄漏到 CLI。
- `config`、`storage/auth` 与显式 `update` 仍可由对应 CLI capability/command 直接依赖。

## 3. 目标目录

```text
internal/cli/
├── root.go
├── root_test.go
├── invocation/
│   └── invocation.go
├── client/
│   ├── client.go
│   ├── auth.go
│   ├── client_test.go
│   └── auth_test.go
├── authstore/
│   ├── store.go
│   └── store_test.go
├── result/
│   ├── movie.go
│   ├── movie_test.go
│   ├── magnet.go
│   ├── magnet_test.go
│   ├── named.go
│   └── named_test.go
├── entity/
│   ├── entity.go
│   └── entity_test.go
└── commands/*
```

最终删除：

```text
internal/cli/app/
internal/cli/movie/
internal/cli/magnet/
```

## 4. 调用期数据

`internal/cli/invocation` 固定提供：

```go
type RootOptions struct {
	Proxy string
	Host  string
}

type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

func NewStreams(in io.Reader, out, err io.Writer) *Streams
```

规则：

- `RootOptions` 只作为 Cobra persistent flags 的目标和 client/update 配置输入。
- `Streams` 只保存调用方提供的 reader/writer，不包装、缓存或替换 IO。
- 该包不导入 config、auth、SDK、Cobra 或 update。
- `root.go` 创建两个对象并按命令需要传入；不合并为具有方法的 Invocation/Session。

## 5. Client 与认证能力

### 5.1 `internal/cli/authstore`

固定接口：

```go
func Open() (*auth.FileStore, *auth.Store, error)
```

实现保持当前顺序：解析默认 auth path，确保配置目录存在，再调用 `auth.Open`。不得吞掉路径、目录或文件错误。`commands/auth` 与 `client` 共同使用该入口，不各自复制默认路径逻辑。

### 5.2 `internal/cli/client`

固定接口：

```go
func New(options *invocation.RootOptions, token string) (*javdb.Client, error)

func WithRequiredAuth(
	options *invocation.RootOptions,
	errOut io.Writer,
	fn func(*javdb.Client) error,
) error

func WithOptionalAuth(
	options *invocation.RootOptions,
	errOut io.Writer,
	fn func(*javdb.Client) error,
) error
```

`New` 吸收现有 `LoadRuntime` 与 `NewClient`：

- 读取 config path 与文件。
- 校验命令行 host。
- 按 flag > environment > file > defaults 解析 runtime。
- 在缺失 device UUID 时沿用现有 `LoadOrCreateDeviceUUID` 路径和错误处理。
- 使用 `javdb.WithHost/WithProxy/WithToken/WithDeviceUUID/WithLang` 构造 client。

`WithRequiredAuth` 从现有 `WithAuthedClient` 原样迁移并改名：

- 无默认账户、空 token、禁用 auto-relogin、无保存密码的错误文本保持。
- 首次调用只在 `AuthRequired` 时进入自动重登。
- 重登成功后更新 store、commit token、设置新 token，并只重试一次回调。
- stderr 提示保持；`errOut == nil` 时不写提示。

`WithOptionalAuth` 从现有 `WithOptionalAuthClient` 原样迁移并改名：

- 无可用默认 token 时匿名执行一次。
- 有 token 且回调返回 `AuthRequired` 时，提示后用匿名 client 重试一次。
- 非认证错误直接返回，不重试。
- 保持当前 auth store 无法提供默认 token 时的匿名行为，不新增其他 fallback。

## 6. Update 依赖归属

`internal/cli/app/update.go` 的能力迁入 `internal/cli/commands/update`：

- `resolveProxy` 使用 `RootOptions.Proxy` 与现有 config/settings 优先级，只服务 GitHub Release 流程。
- `newProductionCoordinator` 组装 release HTTP client、GitHub release client、source detector、command runner 与 archive installer。
- `buildinfo.Current()` 由 update 命令直接调用，不再经共享 facade。
- 新 helper 保持未导出；唯一生产消费者仍是 `update.New`。
- `update.New` 继续先校验 `--json` 必须配合 `--check`，再解析配置或创建网络依赖。

## 7. 统一结果投影

`internal/cli/result` 是纯结果层。三个领域仍分文件，避免单文件聚集不同映射规则。

### 7.1 Movie

```go
type MovieRow struct {
	Number      string
	ID          string
	Title       string
	ReleaseDate string
}

func (r MovieRow) Line() string
func ProjectMovie(item map[string]any) MovieRow
func ProjectMovies(items []map[string]any) []MovieRow
func FilterMoviesWithMagnets(items []map[string]any) []map[string]any
```

保留 float64 ID 截断、可选 release date、输入顺序和缺失 `magnets_count` 时保留记录的语义。

### 7.2 Magnet

```go
type MagnetRow struct {
	Name      string
	Size      string
	Flags     []string
	CreatedAt string
	Hash      string
}

func (r MagnetRow) Line() string
func (r MagnetRow) HashLine() string
func ProjectMagnet(item map[string]any) MagnetRow
func ProjectMagnets(items []map[string]any) []MagnetRow
```

名称 fallback、日期前十字符、cnsub/hd 顺序、MiB 到 MB/GB 格式和 hash URI 文本保持。现有 `Flags`、`FormatSize` 改为未导出 helper；命令只依赖投影结果。

### 7.3 Named

```go
type NamedRow struct {
	ID       string
	Name     string
	Count    any
	HasCount bool
}

func (r NamedRow) Line() string
func ProjectNamed(item map[string]any) NamedRow
func ProjectNamedAll(items []map[string]any) []NamedRow
```

`name_zht` 优先、`name` fallback、`videos_count` 优先于 `movies_count` 以及可选 count 列保持。

`result` 不接收 writer，不输出空列表文案，不编码 JSON，不创建命令，不调用 SDK。命令内现有 `writeMovieRows`、`writeJSON` 等继续由各命令 owner 维护。

## 8. Entity 收窄

`internal/cli/entity` 只保留：

```go
type Options struct { /* 现有字段不变 */ }
type Result struct { /* 现有字段不变 */ }
func Execute(ctx context.Context, client *javdb.Client, kind, ref string, options Options) (Result, error)
```

- `Execute` 改用 `result.FilterMoviesWithMagnets`。
- `NamedRow`、`ProjectNamed`、`ProjectNamedAll` 移至 `result`。
- 实体 metadata 失败降级为 `{"id": eid}` 的现有行为保持，不借重构改变错误策略。
- actor/series/maker/director/code/list 继续各自拥有真实 Cobra 定义，只共享 `Execute`。

## 9. 命令迁移

- 所有现有 `app.Flags` 改为 `invocation.RootOptions`。
- 所有现有 `app.IO`/`app.NewIO` 改为 `invocation.Streams`/`invocation.NewStreams`。
- 匿名命令使用 `client.New(options, "")`。
- required auth 命令使用 `client.WithRequiredAuth(options, streams.Err, fn)`。
- optional auth 命令使用 `client.WithOptionalAuth(options, streams.Err, fn)`。
- `commands/auth` 使用 `authstore.Open` 与 `client.New`，不通过 client 包访问 store。
- movie/magnet/named 调用点改用 `result` 的领域前缀符号。
- 不改变任何命令的 `Use`、`Short`、`Args`、flag、RunE 校验顺序、context 选择或输出代码。

迁移完成后，用 `gopls references`、`rg` 与 `go list` 确认旧符号和旧 import path 无引用，再删除旧目录；不得留下转发 wrapper。

## 10. 测试 owner

### 10.1 根契约

最终将 `contract_test.go` 收敛为唯一的 `root_test.go`，只保留：

- 根 help 完整字面与最终命令集合。
- 关键命令组在最终树中的完整 help 契约。
- `--host`、`--proxy` 的默认值与 usage。
- 未知命令、缺参数等 CLI-wide stderr/stdout/exit-code 契约。
- `New`/`Run` 的最终装配行为。

### 10.2 命令专属测试

根目录测试按下列 owner 迁移或与已有覆盖去重：

| 根测试 | Owner |
|---|---|
| `comments_cmd_test.go` | `commands/comments` |
| `detail_test.go` | `commands/detail` |
| `download_cmd_test.go` | `commands/download` |
| `entity_cmd_test.go` | actor/series/maker/director/code/list 各包 |
| `lists_cmd_test.go` | `commands/lists` |
| `magnets_cmd_test.go` | `commands/magnets` |
| `rankings_cmd_test.go` | `commands/rankings` 与 `commands/top250` |
| `search_cmd_test.go` | `commands/search` |
| `tags_browse_cmd_test.go` | `commands/tags` 与 `commands/browse` |
| `update_cmd_test.go` | `commands/update` |
| `user_cmd_test.go` | watched/want/recent/collections/mark/unmark 与 `client` |
| `version_cmd_test.go` | `commands/version` |

先核对各 owner 的现有断言：相同行为不重复复制，根测试独有的 flag/help/前置校验才迁入。`TestAuthRequiredMessageWithoutAutoRelogin` 这种只检查错误类型有字符串的弱断言删除，由 `client.WithRequiredAuth` 的真实控制流测试替代。

### 10.3 新能力测试

- `client_test.go`：配置/host 解析、空 token client 与 device UUID 行为。
- `auth_test.go`：匿名执行、token fallback、非认证错误、无账号、空 token、auto-relogin 禁用、无密码、重登并持久化 token。
- `authstore/store_test.go`：隔离 HOME 下的默认路径、目录与 store 打开，不接触真实用户文件。
- `result/*_test.go`：原 movie/magnet/named 的全部投影、格式、顺序、数值变体和缺失字段语义。
- `entity_test.go`：只保留解析、tag、单页/全页、过滤、metadata 降级和错误传播。
- `commands/update/update_test.go` 与 `commands/version/version_test.go` 接管当前根行为测试。

## 11. 错误与安全

- 迁移现有错误文本和 wrapping，不借机统一文案。
- 不添加 broad catch、空结果伪成功、隐藏 fallback 或默认成功。
- 不输出 password、JWT、auth 文件内容、token 或本机配置。
- 测试必须重定向 HOME/USERPROFILE 等平台 home 变量到 `t.TempDir()`。
- 不运行真实登录、真实 API、在线 update 或外部文件下载。

## 12. 文档与门禁

内部边界变化需要同步：

- `docs/maintainers/architecture.md`
- `docs/maintainers/development.md`
- `docs/maintainers/adr/0001-public-facade-and-domain-layout.md`
- `scripts/test-architecture.sh`

门禁最终固定：

- `invocation/client/authstore/result/entity` 与全部命令目录存在。
- `app/movie/magnet` 不存在。
- CLI 根目录只有 `root.go` 与 `root_test.go` 两个 Go 文件。
- `result` 不依赖 CLI 命令、SDK、config、storage、update 或协议实现；允许 stdlib 与 `internal/common/scalar`。
- `invocation` 只依赖 stdlib IO。
- command-to-command 依赖继续禁止；CLI 不直接依赖 App API/protocol。
- 每个命令目录仍有与目录同名的主文件。

`goal-1` 保持历史原样。公开 CLI/SDK 契约未变，因此不修改 README、双语用户文档、产品 skill 或 changelog。

## 13. 验证

实施开始先重新记录 HEAD、dirty manifest、Go/gopls 版本与基线。修改后按以下顺序验证：

1. 对所有改动 Go 文件执行 `gopls check`，确认无诊断。
2. 运行 `go test ./internal/cli/result ./internal/cli/authstore ./internal/cli/client ./internal/cli/entity`。
3. 运行 `go test ./internal/cli/...`。
4. 运行 `go test ./...`。
5. 运行 `go test -race ./...`。
6. 运行 `go vet ./...`。
7. 运行 `sh scripts/build.sh`。
8. 运行 `sh scripts/test-architecture.sh` 与 `sh scripts/test-documentation.sh`。
9. 运行 `gofmt -l` 检查、`git diff --check` 和旧路径/旧符号零引用检查。
10. 用构建产物离线冒烟 `--help` 与 `version --json`，不得执行远程命令。

当前 shell 的 Go 1.26.3/GOROOT 1.26.4 混用会导致假失败。验证时使用现有的：

```text
/Users/flanchan/.local/share/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.4.darwin-arm64/bin/go
```

若该路径在实施时变化，先报告环境缺口，不安装或切换系统工具链，待用户授权后处理。

## 14. 完成条件

- 目标目录、接口和依赖方向与本计划一致，无兼容 wrapper。
- 根测试 owner、命令测试 owner 与纯投影测试 owner 清晰，现有契约覆盖不降低。
- 所有公开行为与内部错误、安全约束保持。
- LSP、聚焦测试、全仓 test/race/vet/build、架构和文档门禁通过。
- 维护者文档与最终代码一致，public docs 无无关改动。
- 未触碰用户无关文件，未 commit、push、发布或访问真实远程服务。
