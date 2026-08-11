# v0.5.1 — 2026-08-09

## 新增

- 为 `rankings movies`、`rankings actors`、`rankings playback` 与 `top250` 命令新增 `--json` 输出，保留 `--has-magnets` 过滤，并提供稳定的 `movies`/`actors` 结果键。 ([#14](https://github.com/FlanChanXwO/javdb-cli/pull/14))

## 修复

- 修复影片榜与播放榜的查询参数映射，将 `censored`、`uncensored`、`western`、`fc2` 等文本分区转换为 App API 所需的数字值，并统一各排行接口的周期归一化。 ([#13](https://github.com/FlanChanXwO/javdb-cli/pull/13))

## 新贡献者

- [@kanoshiou](https://github.com/kanoshiou) 在 [#13](https://github.com/FlanChanXwO/javdb-cli/pull/13) 中完成首次贡献。

**完整变更**：[v0.5.0...v0.5.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.0...v0.5.1)
