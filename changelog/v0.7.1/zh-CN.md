# v0.7.1 — 2026-08-14

## 破坏性变更

- 管道命令默认输出人类可读文本，移除冗余的 `--text` 与原 `--jsonl` 参数，并提供显式的 `--json` 和 `--ndjson` 输出模式。 ([#26](https://github.com/FlanChanXwO/javdb-cli/pull/26))

## 修复

- 将 operator skill 元数据同步到 v0.7.1，使不可变 tag 对应的 Release 通过正常交接链路向 ClawHub 发布精确版本。 ([#27](https://github.com/FlanChanXwO/javdb-cli/pull/27))

**完整变更**：[v0.7.0...v0.7.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.0...v0.7.1)
