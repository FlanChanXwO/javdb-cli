# v0.7.1 — 2026-08-14

## 破坏性变更

- 管道命令默认输出人类可读文本。需要管道信封时，请将 `--jsonl` 改为 `--ndjson`；`--json` 保留聚合输出，冗余的 `--text` 参数已移除。 ([#26](https://github.com/FlanChanXwO/javdb-cli/pull/26))

## 修复

- 修复 operator skill 的版本元数据，使 ClawHub 可分发对应的 v0.7.1 skill。 ([#27](https://github.com/FlanChanXwO/javdb-cli/pull/27))

**完整变更**：[v0.7.0...v0.7.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.0...v0.7.1)
