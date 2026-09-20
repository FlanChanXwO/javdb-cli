# v0.8.1 — 2026-09-20

## 变更

- 泛化 README、CLI 参考/帮助与 Agent skill 中的影片示例，避免公开文档绑定具体媒体标识。 ([#51](https://github.com/FlanChanXwO/javdb-cli/pull/51))

## 修复

- 在 Fast Start MP4 remux 过程中归一化 HLS segment 时间戳重置，在音频帧先于视频出现时保持 segment 内音视频时间关系，并使用正确的 FullBox 结构写入 AAC `esds`，提高播放可靠性。 ([#50](https://github.com/FlanChanXwO/javdb-cli/pull/50))

**完整变更**：[v0.8.0...v0.8.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.0...v0.8.1)
