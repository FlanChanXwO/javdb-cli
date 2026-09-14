# Goal 1 计划：CLI contract correctness

## 目标与来源

本 goal 执行用户指定的计划：

<https://github.com/FlanChanXwO/javdb-cli/blob/fix/cli-contract-correctness/docs/plans/2026-09-12-cli-contract-correctness.md>

目标是修复 CLI 与 pipeline 合约缺陷：list envelope 错误、集合结果未 fan-out、显式输出 flag 被 TTY 快捷路径绕过、共享空输入诊断错误、非法 filter 被静默接受、`lists --json` 重复鉴权请求，以及负的 magnet 最小尺寸被接受。

计划明确不处理 movie-number resolver 语义、`download` 重命名、`assets`、`mark`/`unmark`、版本与 release notes、下载器集成，也不修改其他并行分支拥有的源码或文档段落。

## 工作区与分支

- 已按 `using-git-worktrees` 建立隔离 worktree：`/Users/flanchan/Developer/Projects/GithubProjects/javdb-cli/.worktrees/cli-contract-correctness`。
- 实现分支：`codex/cli-contract-correctness`，基于远端 `origin/fix/cli-contract-correctness`。
- 计划来源分支的 HEAD 为 `c3af1bc`；主仓库额外提交了仅用于忽略本地 worktree 的 `chore: ignore local git worktrees`。
- goal 控制文件位于主项目根目录 `goal-1/`，实现、测试与提交全部在上述隔离 worktree 中完成。后续每轮必须显式使用该 worktree 路径。
- 不删除、不覆盖用户已有工作；不使用 `reset --hard`、force push 或删除分支。

## 需求拆解

1. `lists related` 遇到带 ID 的 movie envelope 时直接使用 envelope ID，不把它当打印出的 movie number 再解析；输出 list envelope 时使用 list 自身 ID。
2. `lists search` 与 `lists related` 在 pipeline/NDJSON 模式按结果逐个输出 `kind=list` envelope，保留 `data.list`、正确的 `id` 与 `ref`；human 与单个原始输入的 legacy JSON 保持兼容。
3. `collections --ndjson` 按五类 selector 映射到 singular stable kind，每个实体一个 envelope，实体放在 `data.entity`，使用 `result.ProjectNamed` 投影。
4. `config get --json/--ndjson` 在无 key、TTY 输入时优先于 human `printAll` 路径，输出标准 `config_key` envelope 并沿用脱敏逻辑。
5. `BatchRunner` 空输入诊断使用 runner 的 `Name`，不再复用 search 专属文案。
6. 为 `Producer` 增加可选的 already-produced JSON renderer：先 `Produce` 一次，再渲染；有 renderer 时不调用 `LegacyJSON`，无 renderer 时保持旧顺序。
7. `lists --json` 复用已生成的 envelope 与 page metadata，只发起一次 `MyLists` 请求，不改变 legacy JSON 字段。
8. `search` 与 `lists search` 在发请求前拒绝非 `censored|uncensored|western|fc2|all` 的 zone。
9. `magnets --min-size` 对任何带负号的数值（含负小数与各 suffix）报错；零仍合法。
10. 同步计划所属的 pipeline、JSON、list、collection、filter 文档，不触碰其他分支拥有的段落。

## 实施约束

- 每个行为改动严格走 TDD：先新增能证明缺陷的失败测试（Red），再写最小实现（Green），最后只做必要重构并重跑相关测试。
- 每轮只执行 `tasks.md` 中第一个未完成 task；每三个 task 后执行一次集中检查-debug task。
- 不引入新依赖、不新增 pipeline schema version、不新增 stable kind name、不全局重构 producer。
- 保留 `LegacyJSON` 签名与兼容行为；特别保护 `tags --refresh --json` 的 refresh side effect。
- 代码注释按仓库规则使用中文，文档按既有 locale 与风格编辑，避免无关 Markdown 重排。
- 共享 `BatchRunner` 文案改变时，允许同步既有测试断言（包括被排除命令目录中的测试）；此类同步不得修改 `mark`/`unmark`/`tags` 等排除目录的生产文件或行为。
- 若发现计划与仓库实际接口不一致，先用测试、现有调用方和类型定义收敛；不得扩大到计划外模块。

## 验证策略

按最小相关范围逐级验证：

1. 当前 task 的目标单测或最小复现；必须记录 Red 与 Green 的实际输出。
2. 受影响命令包与 `internal/cli/pipeline/...` 测试。
3. `tags/...` 回归测试，确认 producer JSON 变化没有破坏 refresh。
4. 格式化、静态检查及受影响包构建（按仓库现有命令执行）。
5. 最终执行 `go test ./...` 与 `sh scripts/build.sh`。

计划给出的重点测试范围：

```text
go test ./internal/cli/pipeline/...
go test ./internal/cli/commands/lists/...
go test ./internal/cli/commands/collections/...
go test ./internal/cli/commands/config/...
go test ./internal/cli/commands/search/...
go test ./internal/cli/commands/magnets/...
go test ./internal/cli/commands/tags/...
go test ./...
sh scripts/build.sh
```

输出 envelope 优先使用现有 pipeline validator 验证，不只断言字符串片段；fake server/client 需要断言请求次数与请求前失败行为。

## 风险与回滚

- `Producer.Execute` 是共享路径；任何 JSON 分支改动都可能影响 `tags --refresh --json` 或其他 LegacyJSON producer。Task 1 先锁定调用顺序，Task 3 保持无 renderer 旧路径。
- list 与 collection 同时承担 legacy aggregate 输出和 pipeline fan-out，测试必须明确区分输出模式，避免用新 envelope 改坏兼容 JSON。
- TTY 输入行为可能依赖现有 runner/invocation 抽象；优先复用 `ExecuteWithInputs`，不写第二套 serializer。
- 文档为共享文件，最终只修改计划指定的 pipeline/JSON/list/filter anchors，避免与其他分支的 resolver/download 内容冲突。
- 每个代码 task 独立提交并关联 task 编号；若验证发现回归，优先在对应提交中修正，必要时以反向提交回滚单一 task，保留其他已验证提交。
- 回滚只允许针对本分支提交进行可审计的 `git revert` 或修正提交；不改动主仓库已有提交，不删除 worktree。

## 已完成初始化验证

- `go mod download`：成功。
- worktree 初始 `go test ./...`：成功，退出码 0。
- 初始化阶段未修改业务源码；仅主仓库新增 `.worktrees/` ignore 规则并提交，避免 worktree 内容污染版本控制。

## 完成标准

所有 `tasks.md` task 完成；集中检查与最终审查确认：计划非目标未被触碰，相关测试与构建通过，human/legacy JSON 兼容行为保持，pipeline envelope 可由现有 validator 验证，禁止修改文件清单为空，且没有已知高风险问题。
