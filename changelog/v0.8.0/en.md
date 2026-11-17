# v0.8.0 — 2026-09-20

## Breaking changes

- Replace `javdb download NUMBER --thumbnail/--preview-image/--preview-video` with `javdb assets list ... | javdb assets download ...`. SDK callers move from `DownloadMovieMedia` and its option/result types to `MovieAssets` and `DownloadMovieAsset`. ([#45](https://github.com/FlanChanXwO/javdb-cli/pull/45), [#47](https://github.com/FlanChanXwO/javdb-cli/pull/47))

- Lists/collections NDJSON now emits individual `data.list` / `data.entity` envelopes with required stable IDs instead of aggregate `data.lists` / `data.items`. Human-readable output and aggregate `--json` retain their shapes. ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))

## Added

- Discover and filter thumbnails, preview images, and preview videos with `assets list`, then save selected records with `assets download`. The CLI handles JavDB media headers, wrapped images, and supported HLS decryption, and creates Fast Start MP4 previews without ffmpeg; the SDK exposes `MovieAsset`, `MovieAssets`, and `DownloadMovieAsset`. ([#47](https://github.com/FlanChanXwO/javdb-cli/pull/47))

- `assets list` shows available image dimensions and preview duration in text and JSON/NDJSON (`width`, `height`, `duration`). Configure probing with `assets.probe.enabled` and `assets.probe.concurrency`, or use SDK `ProbeMovieAssets`. Search and detail text output also restores movie duration. ([#48](https://github.com/FlanChanXwO/javdb-cli/pull/48))

- Collection selectors support fan-out, and `config get` supports JSON/NDJSON output. ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))

## Changed

- Source builds and `go install` now require Go 1.27.1, up from 1.26.3. ([#49](https://github.com/FlanChanXwO/javdb-cli/pull/49))

## Fixed

- Movie-number lookup rejects ambiguous or fuzzy-only results, prefers case-insensitive exact matches, and accepts only a unique formatting-equivalent fallback. `mark`/`unmark` honor piped movie IDs, and `mark` validates status before reading stdin. ([#44](https://github.com/FlanChanXwO/javdb-cli/pull/44))

- Invalid collection selectors and non-finite, negative, or overflowing `--min-size` values fail before network requests. Movie/list IDs remain authoritative, and machine envelopes missing stable IDs fail explicitly. ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))

**Full Changelog**: [v0.7.3...v0.8.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.3...v0.8.0)

<!-- internal:
  [#43](https://github.com/FlanChanXwO/javdb-cli/pull/43)
-->
