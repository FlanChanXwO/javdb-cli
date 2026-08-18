# v0.7.2 — 2026-08-18

## 破坏性变更

- 删除隐藏的兼容 `javdb version` 子命令及其 JSON shim。请使用 `javdb --version` 查看人类可读的构建信息；CI、Homebrew、E2E 检查、文档和 operator skill 均已切换到根 flag。 ([#33](https://github.com/FlanChanXwO/javdb-cli/pull/33))

## 新增

- 新增 `search --magnets N`、稳定的磁力排序与筛选、以图搜磁力，以及影片和命名实体的逐项管道记录输出。 ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))

## 变更

- 以版本化双语 changelog 目录作为唯一 release-note 来源，删除旧的 plan/prepare 与 PR metadata 流程；纯文档变更跳过原生 smoke 矩阵，但仍保留聚合的 platform gate。 ([#31](https://github.com/FlanChanXwO/javdb-cli/pull/31))
- 新增自动 PR reviewer/assignee 路由和基于路径的标签，并锁定 GitHub Actions 的提交版本。 ([#33](https://github.com/FlanChanXwO/javdb-cli/pull/33))

## 修复

- 修复非 TTY producer 输出不可稳定消费、上游错误信封被重复处理、重复影片详情请求，以及部分磁力和输出失败未正确报告的问题，并让真实 API 冒烟检查与输出契约一致。 ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))
- 通过核对精确 merge commit 识别 squash merge 的 PR，避免依赖直接的 commit-to-PR 查询。 ([#30](https://github.com/FlanChanXwO/javdb-cli/pull/30))
- 修复跳过矩阵时 packaged-binary smoke job 名称暴露未展开表达式的问题，并对齐双语 Issue 模板。 ([#32](https://github.com/FlanChanXwO/javdb-cli/pull/32))

## 维护

- 记录创建不可变 release tag 前必须以最终 main 提交执行发布前置检查和来源审计的流程。 ([#30](https://github.com/FlanChanXwO/javdb-cli/pull/30))

**完整变更**：[v0.7.1...v0.7.2](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.1...v0.7.2)
