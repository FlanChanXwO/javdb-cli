# v0.8.1 — 2026-09-20

## Changed

- Generalize public movie examples across the READMEs, CLI reference/help, and agent skill so documentation is not tied to concrete media identifiers. ([#51](https://github.com/FlanChanXwO/javdb-cli/pull/51))

## Fixed

- Normalize HLS segment timestamp resets during Fast Start MP4 remuxing, keep segment audio/video timing aligned when audio frames are encountered before video, and emit AAC `esds` with the correct FullBox layout for reliable playback. ([#50](https://github.com/FlanChanXwO/javdb-cli/pull/50))

**Full Changelog**: [v0.8.0...v0.8.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.0...v0.8.1)
