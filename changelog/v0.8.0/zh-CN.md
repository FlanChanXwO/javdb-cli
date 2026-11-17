# v0.8.0 — 2026-09-20

## 破坏性变更

- 将 `javdb download NUMBER --thumbnail/--preview-image/--preview-video` 迁移为 `javdb assets list ... | javdb assets download ...`；SDK 调用从 `DownloadMovieMedia` 及其选项、结果类型迁移到 `MovieAssets` 与 `DownloadMovieAsset`。 ([#45](https://github.com/FlanChanXwO/javdb-cli/pull/45), [#47](https://github.com/FlanChanXwO/javdb-cli/pull/47))

- 列表与合集的 NDJSON 从聚合 `data.lists` / `data.items` 改为逐记录 `data.list` / `data.entity` 信封，并要求稳定 ID；人类可读输出与聚合 `--json` 的结构保持不变。 ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))

## 新增

- 通过 `assets list` 发现并筛选缩略图、预览图片和预览视频，再用 `assets download` 保存选中记录。CLI 处理 JavDB 媒体请求头、包装图片及受支持的 HLS 解密，无需 ffmpeg 即可生成 Fast Start MP4 预览；SDK 提供 `MovieAsset`、`MovieAssets` 与 `DownloadMovieAsset`。 ([#47](https://github.com/FlanChanXwO/javdb-cli/pull/47))

- `assets list` 的文本及 JSON/NDJSON 输出增加可用的图片尺寸和预览时长（`width`、`height`、`duration`）。可通过 `assets.probe.enabled`、`assets.probe.concurrency` 配置探测，或使用 SDK `ProbeMovieAssets`；搜索和详情文本输出也恢复影片时长。 ([#48](https://github.com/FlanChanXwO/javdb-cli/pull/48))

- 合集 selector 支持逐项展开，`config get` 新增 JSON/NDJSON 输出。 ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))

## 变更

- 源码构建与 `go install` 的 Go 版本要求从 1.26.3 提升到 1.27.1。 ([#49](https://github.com/FlanChanXwO/javdb-cli/pull/49))

## 修复

- 番号查询拒绝歧义或仅模糊匹配的结果，优先使用大小写不敏感的完整匹配，仅接受唯一的格式等价回退；`mark`/`unmark` 正确使用管道 movie ID，`mark` 在读取 stdin 前校验状态参数。 ([#44](https://github.com/FlanChanXwO/javdb-cli/pull/44))

- 非法合集 selector，以及非有限值、负数或溢出的 `--min-size` 在联网前报错；保留权威 movie/list ID，缺少稳定 ID 的机器信封显式失败。 ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))

**完整变更**：[v0.7.3...v0.8.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.3...v0.8.0)

<!-- internal:
  [#43](https://github.com/FlanChanXwO/javdb-cli/pull/43)
-->
