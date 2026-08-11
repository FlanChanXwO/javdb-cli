# Goal 1 计划：按真实命令与领域边界重整 Go 目录

## 目标

以当前工作树和已验证的 origin/main@9c3ee65 语义为基线，重新整理 Go 源码目录：

- internal/cli/commands/ 只按真实 CLI 命令拆分，不使用 catalog、user、account 等非命令目录；
- 命令目录的主实现文件使用目录同名文件，例如 commands/search/search.go，不使用泛化的 command.go；
- 删除内部兼容 facade/compat 层，保留 sdk/ 作为唯一公开 Go SDK 边界；
- 删除 cli/output、cli/printer、cli/render 这类横切展示包；
- 跨命令只共享有明确语义的 internal/cli/movie、internal/cli/entity；
- internal/common 只提供分层的纯 JSON/标量转换，不承载 CLI 文案、输出写入或业务过滤；
- 保留 App API endpoint、update、storage、release-note 的现有职责拆分，不把多个子包重新压成单一大文件。

本计划只改变内部目录、包边界和内部调用路径，不改变公开 SDK、CLI 命令、flag、文本输出或 JSON 输出契约。

## 不可违背的边界

- 当前工作树中的用户改动全部保留；不使用 reset、宽范围 checkout、删除用户文件或覆盖式恢复。
- 不创建提交、分支、发布或推送；本计划只描述实施步骤。
- 不新增无证据的 timeout、重试、截断、静默 fallback、错误吞咽或数据限制。
- CLI 命令只能通过 sdk/ 访问远程 JavDB；不得直接导入 internal/javdb/appapi 或 internal/javdb/protocol/*。
- internal/cli/movie 和 internal/cli/entity 只提供影片/实体领域的纯结果处理和行数据投影，不接受泛化的任意输出请求。
- internal/common 根目录不建立 Go 包；代码必须放入 common/jsonx 或 common/scalar。

## 最终目标目录

~~~text
.
├── cmd/javdb/main.go
├── sdk/
│   ├── browse.go
│   ├── client.go
│   ├── client_test.go
│   ├── entity.go
│   ├── errors.go
│   ├── lists.go
│   ├── magnets.go
│   ├── movie.go
│   ├── movie_test.go
│   ├── public_api_test.go
│   ├── rankings.go
│   ├── search.go
│   └── user.go
├── internal/
│   ├── buildinfo/{buildinfo.go,buildinfo_test.go}
│   ├── common/
│   │   ├── jsonx/{jsonx_test.go,object_array.go,object_slice.go,raw_string.go}
│   │   └── scalar/{number.go,scalar_test.go,string.go}
│   ├── cli/
│   │   ├── root.go
│   │   ├── root_test.go
│   │   ├── app/{app.go,auth.go,auth_test.go,update.go}
│   │   ├── movie/{movie.go,movie_test.go}
│   │   ├── entity/{entity.go,entity_test.go}
│   │   └── commands/
│   │       ├── auth/{auth.go,auth_test.go,prompt.go,prompt_test.go}
│   │       ├── config/{config.go,config_test.go}
│   │       ├── search/{search.go,search_test.go}
│   │       ├── detail/{detail.go,detail_lines.go,detail_test.go}
│   │       ├── comments/{comments.go,comments_test.go}
│   │       ├── magnets/{magnets.go,magnets_lines.go,magnets_test.go}
│   │       ├── download/{download.go,download_test.go}
│   │       ├── tags/{tags.go,tags_test.go}
│   │       ├── browse/{browse.go,browse_test.go}
│   │       ├── actor/{actor.go,actor_test.go}
│   │       ├── series/{series.go,series_test.go}
│   │       ├── maker/{maker.go,maker_test.go}
│   │       ├── director/{director.go,director_test.go}
│   │       ├── code/{code.go,code_test.go}
│   │       ├── list/{list.go,list_test.go}
│   │       ├── watched/{watched.go,watched_test.go}
│   │       ├── want/{want.go,want_test.go}
│   │       ├── recent/{recent.go,recent_test.go}
│   │       ├── collections/{collections.go,collections_test.go}
│   │       ├── mark/{mark.go,mark_test.go}
│   │       ├── unmark/{unmark.go,unmark_test.go}
│   │       ├── rankings/{actors.go,movies.go,playback.go,rankings.go,rankings_test.go}
│   │       ├── top250/{top250.go,top250_test.go}
│   │       ├── lists/{lists.go,lists_test.go,related.go,search.go,show.go}
│   │       ├── update/{status.go,update.go,update_test.go}
│   │       └── version/{version.go,version_test.go}
│   ├── config/
│   │   ├── paths/paths.go
│   │   └── settings/{settings.go,settings_test.go}
│   ├── javdb/
│   │   ├── appapi/
│   │   │   ├── auth.go
│   │   │   ├── browse.go
│   │   │   ├── client.go
│   │   │   ├── client_test.go
│   │   │   ├── entity.go
│   │   │   ├── lists.go
│   │   │   ├── magnets.go
│   │   │   ├── media.go
│   │   │   ├── movie.go
│   │   │   ├── rankings.go
│   │   │   ├── search.go
│   │   │   ├── types.go
│   │   │   ├── user.go
│   │   │   ├── client/{device.go,transport.go,transport_test.go}
│   │   │   ├── codec/{codec.go,codec_test.go}
│   │   │   ├── endpoint/
│   │   │   │   ├── auth/auth.go
│   │   │   │   ├── browse/{browse.go,masks.go,masks_test.go}
│   │   │   │   ├── entity/{entity.go,entity_test.go}
│   │   │   │   ├── lists/{lists.go,lists_test.go}
│   │   │   │   ├── magnets/{magnets.go,magnets_test.go}
│   │   │   │   ├── movie/{movie.go,movie_test.go,resolve.go,resolve_test.go}
│   │   │   │   ├── rankings/{rankings.go,rankings_test.go}
│   │   │   │   ├── search/{params.go,params_test.go,search.go}
│   │   │   │   └── user/{user.go,user_test.go}
│   │   │   ├── media/{media.go,media_test.go}
│   │   │   └── model/{search.go,types.go}
│   │   └── protocol/
│   │       ├── httpx/client.go
│   │       └── signature/{sign.go,sign_test.go}
│   ├── storage/
│   │   ├── auth/{file.go,model.go,resolve.go,store.go,store_test.go}
│   │   └── tags/{file.go,model.go,resolve.go,store_test.go}
│   └── update/
│       ├── coordinator.go
│       ├── coordinator_test.go
│       ├── interfaces.go
│       ├── archive/{installer.go,installer_test.go}
│       ├── model/types.go
│       ├── process/{binary.go,command.go,name.go,path.go,replace_nonwindows.go,replace_windows.go}
│       ├── release/{http.go,releases.go,releases_test.go,semver.go,semver_test.go}
│       └── source/{source.go,source_test.go}
└── scripts/
    ├── changescope/{documentation_paths.go,main.go,main_test.go}
    ├── internal/releasenotes/
    │   ├── audit/{audit.go,audit_test.go}
    │   ├── document/{document.go,document_test.go,render.go,render_test.go}
    │   ├── github/{client.go,client_test.go}
    │   ├── history/{history.go,history_test.go}
    │   ├── model/model.go
    │   └── prepare/{prepare.go,prepare_test.go}
    ├── login_probe.go
    ├── releasenotes/{main.go,main_test.go}
    ├── build-release.sh
    ├── build.sh
    ├── package-release.sh
    ├── render-homebrew-formula.sh
    ├── test-architecture.sh
    ├── test-documentation.sh
    ├── test-homebrew-formula.sh
    ├── test-package-release.sh
    ├── test-releasenotes.sh
    └── test-workflows.sh
~~~

## 实施步骤

### 1. 建立基线

- 启动并使用 gopls，记录当前 Go 包、符号、引用和诊断。
- 记录当前 CLI 命令树、注册顺序、flag、文本输出和 JSON 输出。
- 记录 dirty worktree，所有后续修改只覆盖本计划目标文件。

### 2. 迁移纯共享能力

- 将 internal/shared/values 的四类能力迁移到 common/jsonx 与 common/scalar。
- 保持 ObjectArray、ObjectSlice、String、Int64 的既有边界语义。
- RawString 归入 common/jsonx；JSON 编码写出仍由命令直接使用 encoding/json，不建立通用 writer。
- CLI 的 has-magnets、磁力大小、truthy、评论字段降级留在对应命令或 movie/entity 包。
- 删除 internal/shared/values 及其旧引用，补齐 common 子包测试。

### 3. 重整 CLI

- 将 cli/root/root.go 的真实根实现移回 internal/cli/root.go。
- 删除 internal/cli/root、internal/cli/facade.go 和 internal/cli/input。
- 将 prompt.go 归入 commands/auth；将认证测试归入实际 app/auth 或 auth 命令 owner。
- 将 commands/catalog 拆为 search、detail、comments、magnets、download、tags、browse、actor、series、maker、director、code、list。
- 将 commands/user 拆为 watched、want、recent、collections、mark、unmark。
- 将 commands/account 重命名为 commands/auth。
- 将命令主实现文件命名为目录名；新增文件只使用具体职责名称，不使用 command.go、output.go、printer.go 等泛化文件名。
- 将原 output 下的实现归入真实命令；跨多个命令重复的影片/实体结果处理只保留在 internal/cli/movie 和 internal/cli/entity。
- movie/entity 只返回纯结果行/投影或执行纯过滤，不直接成为通用输出包，不依赖 command 包。
- 将原 CLI 根测试按被测命令、movie/entity、app、root 重新归位。

### 4. 重整 App API

- 保留 client、codec、media、model 与 endpoint/{auth,browse,entity,lists,magnets,movie,rankings,search,user}。
- 将 appapi 根 facade.go 改为真实 Client 组合层，按 auth、browse、entity、lists、magnets、movie、rankings、search、user、media 分文件。
- 删除 appapi/facade.go 和 facade_test.go；根包只保留真实 Client、类型契约和 capability 方法，不保留旧路径转发器。
- 保持 sdk 的公开方法、类型和错误契约不变；SDK 仍是唯一公开 facade。

### 5. 重整 config、update、storage

- 删除 internal/config 根 facade；调用方直接依赖 config/paths 与 config/settings。
- 保持配置优先级、文件格式、路径、权限和环境变量语义。
- 保留 update/coordinator.go 为真实协调器；用 interfaces.go 定义协调器依赖，不再通过 facade alias 暴露子包类型。
- 保留 update/model、archive、release、source、process 的文件和测试拆分，删除 update/facade.go、facade_test.go 与根 alias types.go。
- auth、tags 存储目录保持现有 model/file/resolve/store 分工。

### 6. 重整 release-note 脚本

- 保留 scripts/releasenotes/main.go 作为命令分派入口。
- 删除 scripts/releasenotes/compat.go。
- 将原 package-main 大测试拆回 audit、document、github、history、prepare 的真实 owner；main_test.go 只保留入口分派测试。
- 保留 changescope、login_probe.go 与现有构建/测试脚本。

### 7. 同步维护者文档和架构门禁

- 更新 docs/maintainers/architecture.md 与 docs/maintainers/development.md 的目录地图、依赖方向和包职责。
- 更新 docs/maintainers/adr/0001-public-facade-and-domain-layout.md，删除内部 facade/compat 是既定边界的表述。
- 更新 scripts/test-architecture.sh，检查真实命令目录、禁止旧 facade/output/printer/render/catalog/user/account 路径，并验证依赖方向。

## 测试与验收

每个迁移阶段先执行对应包测试，再执行：

~~~bash
go test ./...
go vet ./...
sh scripts/build.sh
sh scripts/test-architecture.sh
sh scripts/test-documentation.sh
sh scripts/test-releasenotes.sh
sh scripts/test-package-release.sh
sh scripts/test-homebrew-formula.sh
sh scripts/test-workflows.sh
~~~

验收必须确认：

- go list ./... 不再出现 internal/cli/root、internal/config 根包或 internal/shared/values；
- 不存在 internal/cli/output、internal/cli/printer、internal/cli/render、internal/cli/commands/catalog、internal/cli/commands/user、internal/cli/commands/account；
- commands/ 下每个目录都对应真实命令或真实命令组；
- 命令主文件使用目录同名文件；
- CLI 命令注册顺序、flag、帮助文本、错误码、文本输出和 JSON 字节语义不变；
- SDK 的公开 import path、导出类型、方法签名和错误匹配不变；
- 测试不再集中在 internal/cli 根目录或 package-main 兼容 wrapper；
- gopls 无新增诊断，go test、go vet、构建和架构门禁全部通过。

## 明确删除的路径

~~~text
internal/cli/facade.go
internal/cli/root/
internal/cli/input/
internal/cli/output/
internal/cli/printer/
internal/cli/render/
internal/cli/commands/catalog/
internal/cli/commands/user/
internal/cli/commands/account/
internal/config/facade.go
internal/config/facade_test.go
internal/javdb/appapi/facade.go
internal/javdb/appapi/facade_test.go
internal/update/facade.go
internal/update/facade_test.go
internal/update/types.go
internal/shared/values/
scripts/releasenotes/compat.go
~~~

## 默认假设

- buildinfo 继续保留为独立包；它负责 linker 注入的 version、commit、build date 和开发构建判断，不并入 version 命令或 update。
- 不建立 internal/cli/dto：当前共享内容包含领域过滤和结果投影，不是跨边界传输对象；若未来引入稳定 typed DTO，再单独设计 dto/movie 与 dto/entity，并禁止其承担 IO 或 CLI 文案。
- 不增加新的公开 SDK 类型或方法；所有新增包均为 internal。
- README、用户 CLI/SDK 文档、changelog 和 skills 只有在用户可见契约改变时才更新；本次结构迁移至少更新维护者架构文档和架构门禁。
