# v0.7.2 — 2026-08-16

## 新增

- 新增 `search --magnets N`、稳定的磁力排序与筛选、以图搜磁力，以及影片和命名实体的逐项管道记录输出。 ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))

## 修复

- 修复非 TTY producer 输出不可稳定消费、上游错误信封被重复处理、重复影片详情请求，以及部分磁力和输出失败未正确报告的问题，并让真实 API 冒烟检查与输出契约一致。 ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))

## 维护

- 让 release-note 审计通过核对精确 merge commit 识别 squash merge 的 PR，并在 project skills 和维护者文档中补充合并后的发布前置流程。 ([#30](https://github.com/FlanChanXwO/javdb-cli/pull/30))

**完整变更**：[v0.7.1...v0.7.2](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.1...v0.7.2)
