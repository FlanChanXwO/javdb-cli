# Goal 1 输入

## 用户请求

将已经讨论并确认的第二阶段 Go 目录重整方案写入 `goal-1/`，但本轮只写计划，不执行任何业务代码、测试、构建、发布或提交操作。

## 目标

以当前 dirty worktree 为唯一实现与行为基线，移除上一阶段为了迁移而建立的内部 facade/compat，按真实 CLI 命令和明确领域职责重整 Go 目录，同时保持公开 SDK 与 CLI 用户契约不变。

## 已确认决策

- 重置 `goal-1`，旧版 `input.md`、`plan.md`、`tasks.md` 原样归档到 `goal-1/archive/initial-domainization/`。
- 当前 dirty worktree 是行为与迁移基线；`9c3ee65` 只作为历史参考，不再称为当前 `origin/main`。
- 内部 import path 和兼容入口允许破坏，不为 CLI、config、update、release-note 的旧内部调用方保留 facade。
- 公开 SDK 不变，继续保留 `Client.API()` 与 taxonomy 方法当前返回类型。
- App API 使用 `endpoint/*` capability service；优先选择可维护性最好、重复代码最少的组合方式。
- 根 `appapi.Client` 保留是因为公开 `Client.API()` 返回该类型，但它必须是真实组合层，不是手写方法转发 facade。
- `commands/` 下只放真实命令或真实命令组；命令主文件与目录同名。
- 六个实体命令各自拥有真实 Cobra 定义，共享 `internal/cli/entity` 的查询用例与纯投影，不保留兼容 wrapper。
- `internal/cli/movie` 名称保留，但职责严格收窄为影片记录的纯投影与过滤。
- 磁力记录的跨命令纯投影放入独立 `internal/cli/magnet`；这不改变 `detail --magnets` 或 `magnets` 的用户行为。
- JSON 字节契约由 `internal/common/jsonx` 的纯编码 helper 统一保证；helper 不接收 writer，也不成为通用输出包。
- 本轮以及后续 goal 默认不自动 commit、建分支、push、发布或运行真实凭据/真实 API。

## 不在本目标内

- 不新增 CLI 命令、flag 或 SDK 能力。
- 不修改公开 CLI/SDK 文档、README、skill 或 changelog，除非实施中发现公开契约实际发生变化；若发生，应视为偏离并先修复实现，而不是接受文档变更。
- 不新增 timeout、重试、截断、静默 fallback、错误吞咽或无证据的数据限制。
- 不回退、覆盖或删除与本目标无关的用户工作树内容。
