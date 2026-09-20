# v0.8.0 — 2026-09-20

## 新增

- 新增两阶段影片资源流程：`javdb assets list NUMBER` 用于发现并筛选缩略图、预览图片和预览视频，`javdb assets download` 消费已选记录并安全写入本地。下载器会处理 JavDB 媒体请求头、XOR 包装图片、已结束的单媒体 MPEG-TS HLS 与 AES-128，并可在不依赖 ffmpeg 的情况下把受支持的 H.264/AAC 预览流 remux 为 Fast Start MP4；公共 SDK 同步提供 `MovieAsset`、`MovieAssets`、`DownloadMovieAsset` 等接口。 ([#47](https://github.com/FlanChanXwO/javdb-cli/pull/47))
- 为 `assets list` 新增有界、best-effort 的流式媒体元数据探测：TTY 可显示图片尺寸和预览时长，JSON/NDJSON 可增加 `width`、`height`、`duration`，并可通过 `assets.probe.enabled` / `assets.probe.concurrency` 控制；SDK 新增 `ProbeMovieAssets`。人类可读的 `search` 与 `detail` 输出也恢复正片时长。 ([#48](https://github.com/FlanChanXwO/javdb-cli/pull/48))
- 新增客户端侧合集 selector fan-out，以及 `config get` 的 JSON/NDJSON 机器输出，同时保留文档明确承诺的 legacy 聚合 JSON 与人类可读输出。 ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))

## 变更

- 将源码构建、`go install`、CI 与 Release 构建支持的 Go toolchain 从 1.26.3 提升到 Go 1.27.1；workflow 继续统一从 `go.mod` 读取版本，并在中英文 README / CONTRIBUTING 中明确声明 Go 1.27.1。 ([#49](https://github.com/FlanChanXwO/javdb-cli/pull/49))
- **破坏性变更：** 从 v0.7.3 升级时，需要把 `javdb download NUMBER --thumbnail/--preview-image/--preview-video` 迁移为可组合的 `javdb assets list ... | javdb assets download ...` 流程；公共 SDK 也从 `DownloadMovieMedia` 与 `MovieMediaDownloadOptions` / `MovieMediaDownloadResult` 迁移为 `MovieAssets` + `DownloadMovieAsset`。 ([#45](https://github.com/FlanChanXwO/javdb-cli/pull/45), [#47](https://github.com/FlanChanXwO/javdb-cli/pull/47))
- **破坏性变更：** lists/collections 的 NDJSON 从聚合 `data.lists` / `data.items` payload 迁移为逐记录 `data.list` / `data.entity` envelope，并要求稳定 ID；legacy 人类可读输出和显式聚合 `--json` 的 shape 保持不变。 ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))
- 在中英文 README 中补充 JavDB 官方网站和官方 App GitHub Releases 下载入口。 ([#43](https://github.com/FlanChanXwO/javdb-cli/pull/43))

## 修复

- 防止番号解析在只有模糊结果或存在歧义时静默选择搜索首项：忽略首尾空白，优先接受大小写不敏感的完整匹配，只有唯一且无歧义的格式等价番号才允许回退；同时修复 `mark` / `unmark` 的管道 movie ID 优先级，并避免 `mark` 在 status 校验阶段提前读取 stdin。 ([#44](https://github.com/FlanChanXwO/javdb-cli/pull/44))
- 在创建客户端或发起网络请求前校验合集 selector 与 `magnets --min-size`，拒绝非有限值、负数和溢出尺寸，保留权威 movie/list ID，并让缺少稳定 ID 的机器信封显式失败。 ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))

**完整变更**：[v0.7.3...v0.8.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.3...v0.8.0)
