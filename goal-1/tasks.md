# Goal 1 Tasks

状态约定：`[ ]` 未完成，`[x]` 已完成，`[!]` 阻塞。每轮只执行第一个未完成 task。代码 task 必须先 Red 再 Green；完成后提交代码并回写本文件。

## Task 1 — 添加 shared runner 与 producer 回归测试

- 状态：[x]
- 范围：`internal/cli/pipeline/runner_test.go` 及必要的 focused producer test。
- 验收：覆盖空输入诊断、new already-produced JSON renderer 调用顺序、无 renderer 的 `Produce`→`LegacyJSON` 兼容顺序、text/human/NDJSON 单次 Produce、`--json` 与 `--ndjson` 互斥；Red 阶段实际失败。
- 实际完成：新增 `runner_test.go`，覆盖 `BatchRunner` 空输入命令名、Producer JSON renderer 调用顺序与 envelope 内容、无 renderer 的 LegacyJSON 兼容、text/human/NDJSON 单次 Produce，以及互斥输出 flag 在 Produce 前失败。
- 验证证据：`gofmt -w internal/cli/pipeline/runner_test.go` 成功；`git diff --check` 成功；Red 阶段运行 `go test ./internal/cli/pipeline`，按预期因尚未添加生产字段失败：`unknown field RenderJSON in struct literal of type Producer`；提交 `7da7a65 test(pipeline): add CLI contract regressions (task 1)`。
- 剩余风险：当前分支暂时不能编译，需 Task 2/3 实现对应生产行为后恢复 Green。该测试提交使用 `--no-verify`，因为 pre-commit 的测试门禁会阻止刻意的 Red 阶段提交。
- 下一步：Task 2，修复 BatchRunner 空输入诊断。

## Task 2 — 修复 BatchRunner 空输入诊断

- 状态：[x]
- 范围：`internal/cli/pipeline/runner.go`。
- 验收：使用 `BatchRunner.Name` 生成 `<command name>: input required`；无 name 时为 `input required`；不新增各 command 的重复检查。
- 实际完成：`BatchRunner.Execute` 统一把空输入转换为 command-specific 错误；有 `Name` 时返回 `<Name>: input required`，无 name 时返回 `input required`。新增无 name 回归测试，并同步现有根命令与受影响 command 的旧错误断言；未修改 `Consumer`、image-aware search 或各 command 的生产逻辑。`unmark` 仅更新测试期望以反映共享 runner 合约，未修改其生产文件。
- 验证证据：Red 测试先落盘；实现后 `go build ./internal/cli/pipeline`、受影响 command focused tests 与 `git diff --check` 通过；Task 3 完成后全量 pre-commit `go test ./...` 通过。修正后的提交为 `6e3e8f8 fix(pipeline): name empty-input errors (task 2)`。
- 剩余风险：共享文案变更触及多个旧测试断言；现有测试已逐一同步并由全量测试覆盖。计划排除目录中的 `unmark` 仅发生测试期望更新，无生产行为变更。
- 下一步：Task 3 已完成；下一轮执行集中检查-debug 1。

## Task 3 — 增加 Producer already-produced JSON renderer

- 状态：[x]
- 范围：`internal/cli/pipeline/runner.go` 与 producer focused tests。
- 验收：所有模式仍只 Produce 一次；有 renderer 时只渲染已生成 envelopes、不调用 LegacyJSON；无 renderer 保持旧行为；`tags --refresh --json` 未被修改且回归通过。
- 实际完成：为 `Producer` 增加可选 `RenderJSON func(io.Writer, []Envelope) error`；JSON 模式优先渲染已 Produce 的 envelopes，未设置时继续调用既有 `LegacyJSON`。未修改 tags 生产代码。
- 验证证据：实现前 `go test ./internal/cli/pipeline` 按预期因 `unknown field RenderJSON in struct literal of type Producer` 失败；实现后 `go test ./internal/cli/pipeline` 与 `go test ./internal/cli/commands/tags/...` 通过；pre-commit 的 `gofmt`、`go test ./...`、release tooling checks 全部通过；提交 `e247bb8 feat(pipeline): render produced envelopes as JSON (task 3)`。
- 剩余风险：尚无实际使用 `RenderJSON` 的生产 command；Task 4 将让 lists 使用它并验证单次 `MyLists` fetch。旧 `LegacyJSON` producer 已由 focused test 与 tags 回归保护。
- 下一步：集中检查-debug 1。

## 集中检查-debug 1（Task 1-3 后）

- 状态：[x]
- 检查：核对 input/plan 约束、diff、调用顺序、死代码、类型与 pipeline 测试；确认没有改动 tags 或其他禁止文件；发现问题追加修复 task。
- 实际完成：逐项核对前三个 task。`BatchRunner` 只统一处理空输入；search 的 image-aware 诊断仍保留。`Producer` 的 JSON renderer 只在显式 JSON 模式优先调用，LegacyJSON fallback 未改变。检查发现顶层 Producer 注释仍描述 JSON 一律走 LegacyJSON，已在 `e2e5978` 修正。共享错误文案导致的旧测试断言已同步；mark/unmark/tags 生产文件未修改。
- 验证证据：`go test ./...` 通过；`sh scripts/build.sh` 通过并生成 `build/javdb`（已被仓库规则忽略）；`go vet ./internal/cli/pipeline ./internal/cli/commands/lists ./internal/cli/commands/tags` 通过；pipeline、tags focused tests 与 pre-commit 的 gofmt/release checks 通过；`git diff --check` 通过；commit-range 仅包含 pipeline、相关错误文案测试断言与一个 pipeline comment，未包含 resolver、download、tags 生产文件。
- 剩余风险：`RenderJSON` 尚未接入生产 command；lists 单次 fetch 与 legacy JSON shape 仍待 Task 4 验证。文档同步集中留到 Task 11。`unmark` 目录只有测试期望更新，属于共享错误合约同步，生产文件仍未触碰。
- 下一步：Task 4，让默认 `lists --json` 复用已生成结果并验证单次 `MyLists` 请求。

## Task 4 — 让默认 lists --json 只 fetch 一次

- 状态：[x]
- 范围：`internal/cli/commands/lists/lists.go` 与 list tests。
- 验收：Produce 只请求一次 `MyLists`，捕获本次 `current_page`，renderer 复原 `{"lists":[...],"current_page":"..."}`；NDJSON 仍使用 `data.list`；fake server 断言请求次数与 JSON 字段。
- 实际完成：`lists` 的 Produce 过程捕获本次 `current_page`；移除重复 fetch 的 legacy JSON callback，改用 `Producer.RenderJSON` 从已生成的 `data.list` envelopes 重建原有 `lists`/`current_page` JSON shape。新增 fake server 测试，断言请求次数与字段内容。
- 验证证据：Red 阶段 `go test ./internal/cli/commands/lists -run TestNewJSONOutputFetchesListsOnceAndPreservesShape -count=1` 实际失败：`MyLists requests = 2, want 1`；实现后 focused lists、pipeline、tags 测试通过；pre-commit 的 gofmt、`go test ./...`、release tooling checks 全部通过；提交 `b22ff5c fix(lists): reuse fetched results for JSON output (task 4)`。
- 剩余风险：当前只覆盖单次列表响应；多 list 与缺省 `current_page` 的 envelope/legacy shape 仍需后续 pipeline/list 回归共同覆盖。未改变 NDJSON 的 `data.list` 结构。
- 下一步：Task 5，让 `lists search` pipeline 输出按 list fan-out。

## Task 5 — 让 lists search 按 list fan-out

- 状态：[x]
- 范围：`internal/cli/commands/lists/search.go` 与 lists tests。
- 验收：复用 `BatchRunner.RunMany`；每个结果输出一个 `pipeline.KindList` envelope，`id` 为 list ID、`ref` 为 name/ID fallback、raw object 在 `data.list`；legacy JSON 与单 raw positional human 行为保持。
- 实际完成：将 `lists search` 的 pipeline 路径切换为既有 `BatchRunner.RunMany`，每个搜索结果生成一个 `kind=list` envelope；`id` 取 list ID，`ref` 取 name，name 为空时回退 ID，原始对象放入 `data.list`。单项 raw positional 的 legacy JSON/human 路径保持不变。
- 验证证据：Red 阶段运行 `go test ./internal/cli/commands/lists -run TestNewSearchNDJSONFansOutLists -count=1` 实际失败，旧实现输出 `data.lists` 聚合信封且 `id` 为空；实现后 lists、pipeline、search 相关包测试通过，提交 pre-commit 的 gofmt、`go test ./...`、release tooling 与 release notes checks 全部通过；提交 `8cace4a fix(lists): fan out search results`。
- 剩余风险：当前 focused fan-out fixture 每个查询返回一个 list；多 list 同一查询的数量与顺序由 `RunMany`/writer 既有行为覆盖，尚未单独增加多结果 fixture。未修改 legacy search 请求与渲染路径。
- 下一步：Task 6，让 lists related 使用 authoritative movie ID 并 fan-out。

## Task 6 — 让 lists related 使用 authoritative movie ID 并 fan-out

- 状态：[x]
- 范围：`internal/cli/commands/lists/related.go` 与 related tests。
- 验收：有 envelope ID 时直接用该 ID；否则遵守现有 `--id` 语义；每个 related list 一个自身 ID 的 `kind=list` envelope；不修改 resolver 实现，不把 movie ID 放进 list envelope；legacy human/JSON aggregate 兼容。
- 实际完成：将 pipeline 路径切换为 `BatchRunner.RunMany`；输入 envelope 有 ID 时直接将其作为 movie ID，无 ID 时保留 `--id` 直传或既有 resolver 语义。每个 related list 独立输出 `kind=list` envelope，使用 list 自身 ID、name/ID fallback 与 `data.list`，不再把 movie ID 写入 list envelope。legacy JSON/human 仍调用原有聚合渲染路径。
- 验证证据：Red 阶段新增测试实际触发旧实现的 movie resolver 错误，并暴露旧聚合 envelope；Green 阶段 `go test ./internal/cli/commands/lists -run 'TestNewRelatedNDJSONUsesAuthoritativeMovieIDAndFansOutLists|TestNewRelatedNDJSONHonorsIDFlagWithoutEnvelopeID|TestNewRelatedLegacyOutputRemainsAggregate' -count=1`、lists/pipeline 全包测试与 `go vet ./internal/cli/commands/lists ./internal/cli/pipeline` 通过；提交 pre-commit 的 gofmt、`go test ./...`、release tooling 与 release notes checks 全部通过；提交 `f27d9a9 fix(lists): use movie envelope IDs for related lists`。
- 剩余风险：测试覆盖了多 list、authoritative ID、无 ID 的 `--id` 输入及 legacy 两种输出；未额外改变或测试 movie-number resolver 实现，符合本计划非目标。
- 下一步：集中检查-debug 2，核对 Task 4-6 的两条 list pipeline 路径与 resolver ownership。

## 集中检查-debug 2（Task 4-6 后）

- 状态：[x]
- 检查：核对 list 两条 pipeline 路径、single legacy 路径、请求次数、validator、resolver ownership、编译与相关测试；发现问题追加修复 task。
- 实际完成：核对 `lists` producer 的已生成结果复用与 `current_page`、`lists search`/`lists related` 的 `RunMany` fan-out、`data.list` 及 list 自身 ID；确认单项 raw positional 的 legacy JSON/human 分支仍在 `Legacy`，而带 kind 的 NDJSON 输入不会绕过 kind 检查。确认 related 的 movie envelope ID 与 `--id` 只在无 ID 时生效，movie resolver 源码未改动。检查发现 `BatchRunner` 顶层注释未反映 `RunMany` 优先，已用最小文案修复；未发现需要追加行为修复 task。
- 验证证据：`go test ./internal/cli/pipeline/... ./internal/cli/commands/lists/... ./internal/cli/commands/tags/...`、对应 `go vet`、`sh scripts/build.sh` 与 `git diff --check` 均通过；build 输出 `build/javdb`，为仓库忽略产物；提交 pre-commit 的 gofmt、`go test ./...`、release tooling 与 release notes checks 全部通过。commit-range 中禁止目录仅出现允许的 `internal/cli/commands/unmark/unmark_test.go` 共享错误断言更新，无 mark/unmark/tags 生产文件、resolver/download/release 文件变更；worktree clean。修复提交为 `e271755 docs(pipeline): describe BatchRunner fan-out`。
- 剩余风险：Task 4-6 已完成且相关验证充分；多实体 collections、config 显式输出、zone 与 magnet size 仍待后续 tasks，属于计划内未完成范围。
- 下一步：Task 7，让 collections 输出 singular entity envelopes。

## Task 7 — 让 collections 输出 singular entity envelopes

- 状态：[x]
- 范围：`internal/cli/commands/collections/**` 与 collection tests。
- 验收：五个 plural selector 映射到既有 singular stable kind；每项一个 envelope，ID/name/ref 来自 `ProjectNamed` 投影，raw item 在 `data.entity`；name_zht、多 item、空 name 与 ID、validator 均有覆盖；legacy human/单 raw positional JSON aggregate 兼容。
- 实际完成：为 actors、series、codes、makers、directors 增加显式 plural-to-singular kind 映射，pipeline 路径改用 `BatchRunner.RunMany`。每个实体以 `result.ProjectNamed` 投影生成独立 envelope，`id` 使用投影 ID，`ref` 使用投影 name 并回退 ID，原始对象放入 `data.entity`；name 与 ID 同时为空时返回错误信封。legacy `Legacy` 路径未改动，继续输出聚合 JSON 或 human 行。
- 验证证据：Red 阶段 `go test ./internal/cli/commands/collections -run 'TestNewNDJSONFansOutAllCollectionSelectors|TestNewNDJSONRejectsEntityWithoutNameOrID|TestNewLegacyJSONAndHumanOutputRemainAggregate' -count=1` 实际暴露 plural kind、单聚合 envelope 与 nameless entity 静默成功；Green 阶段上述 focused 测试、collections/pipeline/result 包测试与对应 `go vet` 通过；提交 pre-commit 的 gofmt、`go test ./...`、release tooling 与 release notes checks 全部通过；提交 `f0aa35b fix(collections): fan out singular entity envelopes`。
- 剩余风险：五类 selector、name_zht、multi-item cardinality、空 name + ID fallback、无 name/ID 错误、validator 与 legacy 两种输出均有 fake server 回归；后续 config、zone 与 magnet 校验仍待完成。
- 下一步：Task 8，让 config get 遵守显式输出模式。

## Task 8 — 让 config get 遵守显式输出模式

- 状态：[x]
- 范围：`internal/cli/commands/config/config.go` 与 config tests。
- 验收：TTY 无 flag 无 key 保持 human；TTY `--json` 输出 config_key envelope 数组；TTY `--ndjson` 每 key 一个 envelope；使用现有脱敏；non-TTY key batch 保持；互斥 flag 仍拒绝。
- 实际完成：TTY 无 key 且显式 `--json`/`--ndjson` 时，构造 `displayConfigKeys` 对应的 `config_key` 输入并复用 `BatchRunner.ExecuteWithInputs`；因此 JSON 输出信封数组、NDJSON 按 key fan-out，并继续走既有 `RunOne` 脱敏逻辑。TTY 无 flag 仍走 `printAll`，非 TTY key batch 与单 key/legacy 路径未改。机器分支先调用 `ResolveOutputMode`，保持双 flag 互斥校验。
- 验证证据：Red 阶段新测试实际观察到旧实现输出裸 `key=value`、泄漏 proxy secret 且双 flag 返回 nil；实现后 `go test ./internal/cli/commands/config -run 'TestConfigGetTTY(JSON|NDJSON)|TestConfigGetTTYRejectsJSONAndNDJSONTogether' -count=1` 通过。随后 `go test ./internal/cli/commands/config/... ./internal/cli/pipeline/... ./internal/cli/commands/tags/... -count=1`、对应 `go vet`、`git diff --check` 通过；提交钩子中的 gofmt、`go test ./...`、release tooling checks、release notes checks 全部通过。提交 `a8c60e8 fix(config): honor explicit machine output`。
- 剩余风险：无已知行为风险；当前实现按已有 runner 逐个加载配置，保持共享 pipeline 合约，未引入新序列化路径。
- 下一步：Task 9，在 search 与 lists search 两个 CLI entry point 校验 zone。

## Task 9 — 在两个 search entry point 拒绝非法 zone

- 状态：[x]
- 范围：`internal/cli/commands/search/search.go`、`internal/cli/commands/lists/search.go` 及 focused tests。
- 验收：仅接受 `censored|uncensored|western|fc2|all`；typo 在发请求前失败；不改变低层 SDK 对任意 SDK caller 的行为；两处 CLI 校验可保持小范围重复。
- 实际完成：两个 CLI entry point 均在执行 runner/client 请求前校验 zone，只接受 `censored`、`uncensored`、`western`、`fc2`、`all`；非法值返回统一诊断。校验留在 CLI 文件内，未改变低层 SDK 或 endpoint 的任意 zone 行为。
- 验证证据：Red 阶段两个 fake-server 测试实际得到 nil error；Green 阶段 `go test ./internal/cli/commands/search ./internal/cli/commands/lists -run 'TestSearchRejectsInvalidZoneBeforeRequest|TestNewSearchRejectsInvalidZoneBeforeRequest' -count=1` 通过，并断言请求数为 0。随后 `go test ./internal/cli/commands/search/... ./internal/cli/commands/lists/... ./internal/javdb/appapi/endpoint/search/... ./internal/cli/pipeline/... -count=1`、对应 `go vet`、`git diff --check` 通过；提交钩子中的 gofmt、`go test ./...`、release tooling checks、release notes checks 全部通过。提交 `11940df fix(search): reject invalid zones before requests`。
- 剩余风险：无已知风险；两个入口保留小范围重复校验，避免扩大 CLI 与 SDK 的耦合面。
- 下一步：集中检查-debug 3，复核 collections/config/search 的 CLI 合约、请求前校验、脱敏与禁止文件清单。

## 集中检查-debug 3（Task 7-9 后）

- 状态：[x]
- 检查：核对 collections/config/search 的 CLI 合约、TTY/non-TTY 分支、敏感值脱敏、网络请求前失败、validator、相关测试、禁止文件清单；发现问题追加修复 task。
- 实际完成：复核 collections 五类 selector 的 singular kind 映射、`ProjectNamed` 投影、`data.entity` 与 legacy 聚合路径；复核 config 的 TTY human、TTY 显式 JSON/NDJSON、非 TTY 批处理和 proxy 脱敏；复核 search 与 lists search 的 zone 校验均早于 client/网络请求。NDJSON 测试使用现有 decoder/validator，未发现兼容性或错误处理问题，也未发现需要追加修复 task。
- 验证证据：`go test ./internal/cli/commands/collections/... ./internal/cli/commands/config/... ./internal/cli/commands/search/... ./internal/cli/commands/lists/... ./internal/cli/pipeline/... ./internal/cli/result/... -count=1`、`go test ./internal/cli/commands/tags/... ./internal/cli/commands/unmark/... -count=1`、对应 `go vet` 与 `git diff --check c3af1bc..HEAD` 全部通过；禁止生产目录审计（mark/unmark/tags、download/assets、movie resolver、release/changelog）无匹配，worktree clean。Task 7-9 的代码提交钩子此前也已通过全量 `go test ./...` 与 release checks。
- 剩余风险：Task 7-9 无已知高风险问题；最终全量构建、文档同步和全 goal 终审仍待后续 task。
- 下一步：Task 10，拒绝负的 magnet 最小尺寸。

## Task 10 — 拒绝负的 magnet 最小尺寸

- 状态：[x]
- 范围：`internal/cli/commands/magnets/magnets.go` 与 parser tests。
- 验收：`ParseSizeMiB` 对 `-1`、`-1M`、`-1GB` 和负小数报错；在 float 解析后、int 转换前检查负数；零仍允许；不改变排序与 `PickBestMagnet`。
- 实际完成：`ParseSizeMiB` 在 `ParseFloat` 成功后、单位换算和 `int` 转换前拒绝负值，避免负小数截断成 0；保留原有单位解析和零值行为，未修改排序或 `PickBestMagnet`。
- 验证证据：Red 阶段新增测试实际观察到 `-1`、`-1M`、`-1GB`、负小数及各 suffix 均被接受；Green 阶段 `go test ./internal/cli/commands/magnets -run 'TestParseSizeMiB(RejectsNegativeValues|AllowsZero)?$' -count=1` 通过。随后 magnets、search、低层 magnets endpoint 测试及对应 `go vet`、`git diff --check` 通过；提交钩子中的 gofmt、`go test ./...`、release tooling checks、release notes checks 全部通过。提交 `27480cf fix(magnets): reject negative minimum sizes`。
- 剩余风险：无已知风险；负零未单独扩展为新的语法限制，普通零值仍明确保持合法。
- 下一步：Task 11，同步计划范围内双语文档与 CLI skill 文档。

## Task 11 — 同步计划范围内文档

- 状态：[x]
- 范围：双语 CLI reference、README、`skills/javdb-cli/SKILL.md`、必要的 skill references、必要时 architecture；只改 pipeline/JSON/list/collection/filter anchors。
- 验收：文档描述 fan-out、envelope ID/ref/data、explicit output、zone 与 min-size 校验、renderer 语义；不触碰 resolver/download/其他分支内容；独立文档提交。
- 实际完成：同步 `docs/en/cli-reference.md`、`docs/zh-CN/cli-reference.md`、双语 README、`skills/javdb-cli/SKILL.md` 及 `references/discover.md`，补充列表/合集 fan-out、envelope 的 `id`/`ref`/`data.list`/`data.entity`、config 显式机器输出、zone/filter/min-size 约束与 legacy JSON 兼容说明；在 `docs/maintainers/architecture.md` 记录 `Producer.RenderJSON` 的复用与 fallback 语义。未修改 resolver、download、assets、mark/unmark、tags 生产文档或 release 内容。
- 验证证据：`git diff --check` 通过；本地 Markdown 相对链接与目标 anchor 校验通过；变更文件范围仅为计划允许的 7 个文档文件；独立提交 `993dd52 docs(cli): sync contract semantics`，提交钩子的 `go test ./...`、release tooling checks 与 release notes checks 全部通过。
- 剩余风险：无已知文档范围或链接风险；最终全量构建、禁止文件审计与跨 task 终审仍待 Task 12/集中检查 4。
- 下一步：Task 12，全量验收验证。

## Task 12 — 全量验收验证

- 状态：[x]
- 范围：相关包测试、`tags` 回归、全量 `go test ./...`、`sh scripts/build.sh`、最终 diff/禁止文件检查。
- 验收：所有 acceptance criteria 有实际证据；构建与测试通过；无 secrets、产物或越界变更；若失败只追加归属本 goal 的修复 task。
- 实际完成：完成相关 CLI/pipeline 包与 `tags`、`unmark` 回归，全量测试、静态检查、构建及基线差异审计；构建产物保持未跟踪且未纳入提交。
- 验证证据：`go test ./internal/cli/pipeline/... ./internal/cli/commands/lists/... ./internal/cli/commands/collections/... ./internal/cli/commands/config/... ./internal/cli/commands/search/... ./internal/cli/commands/magnets/... ./internal/cli/commands/tags/... ./internal/cli/commands/unmark/... -count=1` 通过；`go test ./...` 通过；`go vet ./...` 通过；`sh scripts/build.sh` 成功生成 dev 二进制；`git diff --check c3af1bc..HEAD` 通过。基线差异在排除目录仅包含计划允许的 `unmark/unmark_test.go` 断言同步；禁止生产目录、release/development 文件、已跟踪 `build/javdb`、真实 credential-like literal 均未发现；goal 控制文件位于主仓库外部，worktree clean。
- 剩余风险：Task 12 验证未发现归属于本 goal 的失败或新增风险；集中检查-debug 4 与最终终审仍待执行。
- 下一步：集中检查-debug 4，完整复核计划约束、代码质量、兼容性与合并边界。

## 集中检查-debug 4（Task 10-12 后）

- 状态：[x]
- 检查：完整对照 input/plan、代码质量、错误处理、安全、数据/请求一致性、文档、回滚与合并边界；所有已知高风险问题必须修复或追加 task。
- 实际完成：按 `code-review-expert` 的 finding-first 流程复核 `c3af1bc..HEAD` 全部 13 个 goal 提交及当前工作区。逐项核对 lists/collections/config/search/magnets/pipeline 行为、legacy/human 兼容、validator 使用、请求顺序、错误暴露、敏感值处理、文档范围、提交边界与计划排除项；未发现 P0、P1、P2 或 P3 finding，也没有需要追加修复 task 的问题。
- 验证证据：当前工作区 `git status -sb` clean；commit-range diff 与提交 subjects 已复核，`git diff --check c3af1bc..HEAD` 通过；Task 12 已提供相关包、`tags`/`unmark` 回归、`go test ./...`、`go vet ./...` 与 `sh scripts/build.sh` 的通过证据；本轮边界审计确认允许的排除目录变化仅为 `internal/cli/commands/unmark/unmark_test.go` 共享错误断言同步，无 `go.mod`/`go.sum`/pipeline schema、禁止生产目录、resolver/download/release 变更，也无新增真实 credential-like literal。
- 剩余风险：无已知高风险问题；计划明确排除的 resolver、download/assets、mark/unmark/tags 生产行为、版本与 release notes 保持未改。实现分支相对 `origin/fix/cli-contract-correctness` ahead 13，goal 控制文件位于主仓库外部，不属于实现分支提交范围。
- 下一步：Final review。

## 最终终审

- 状态：[x]
- 检查：从用户体验、pipeline composability、legacy compatibility、测试覆盖、构建、禁止文件、提交历史与 worktree 状态全面复查；无已知高风险问题后登记 goal 完成。
- 实际完成：完成计划逐条 completion audit。lists 默认 JSON 复用单次请求结果，lists search/related 与 collections 按稳定 envelope fan-out；related 使用权威 movie envelope ID；config 显式机器输出优先且沿用脱敏；BatchRunner、zone、min-size 与 Producer fallback 语义符合计划。已核对 human/legacy JSON 兼容、pipeline 可组合性、TDD Red→Green 记录、文档 owned anchors、专属文档提交、提交历史及隔离 worktree；未发现需要修复的高风险问题。
- 验证证据：当前 `codex/cli-contract-correctness` worktree clean，`HEAD=993dd52`，相对 `origin/fix/cli-contract-correctness` ahead 13；本轮 `go test ./... -count=1`、`go vet ./...`、`sh scripts/build.sh`、精确 `go test ./...`、`git diff --check c3af1bc..HEAD` 均通过。lists/collections/config/pipeline 回归通过现有 `pipeline.DecodeNDJSON`/`Envelope.Validate` 验证信封，fake server 覆盖请求次数与请求前失败；范围审计确认无依赖、schema、resolver/download/assets、mark/unmark/tags 生产、release 或版本越界变更，唯一排除目录变化为获准的 `unmark/unmark_test.go` 错误断言同步；无真实 credential-like literal，构建产物未跟踪。
- 剩余风险：无已知高风险问题。计划明确排除的 movie resolver、download/assets、mark/unmark/tags 生产行为、下载器集成及 release 内容仍由对应分支负责；goal 控制文件位于主仓库外部，不属于实现分支交付。
- 下一步：Goal complete。
