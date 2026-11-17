# v0.7.2 — 2026-08-18

## 破坏性变更

- 移除隐藏的 `javdb version` 兼容命令及其 JSON 输出；请使用 `javdb --version` 查看构建信息。 ([#33](https://github.com/FlanChanXwO/javdb-cli/pull/33))

## 新增

- 新增 `search --magnets N`，一次完成影片搜索、磁力排序与筛选，并支持以图搜磁力。管道可逐项展开影片与命名实体记录。 ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))

## 修复

- 生产者文本输出使用稳定引用，便于后续命令消费；上游错误、部分磁力失败和输出错误完整报告，并减少重复影片详情请求。 ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))

**完整变更**：[v0.7.1...v0.7.2](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.1...v0.7.2)

<!-- internal:
  [#31](https://github.com/FlanChanXwO/javdb-cli/pull/31)
  [#30](https://github.com/FlanChanXwO/javdb-cli/pull/30)
  [#32](https://github.com/FlanChanXwO/javdb-cli/pull/32)
-->
