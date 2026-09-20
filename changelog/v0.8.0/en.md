# v0.8.0 — 2026-09-20

## Added

- Add a two-stage movie asset workflow: `javdb assets list NUMBER` discovers and filters thumbnails, preview images, and preview videos, while `javdb assets download` consumes the selected records and safely writes them locally. The downloader handles JavDB media request headers, XOR-wrapped images, ended single-media MPEG-TS HLS with AES-128, and can remux supported H.264/AAC previews to Fast Start MP4 without ffmpeg; the public SDK now exposes `MovieAsset`, `MovieAssets`, `DownloadMovieAsset`, and related helpers. ([#47](https://github.com/FlanChanXwO/javdb-cli/pull/47))
- Add bounded, best-effort streaming media metadata probing for `assets list`: TTY output can show image size and preview duration, JSON/NDJSON may include `width`, `height`, and `duration`, `assets.probe.enabled` / `assets.probe.concurrency` control the behavior, and the SDK exposes `ProbeMovieAssets`. Human-readable `search` and `detail` output also restores feature-film duration. ([#48](https://github.com/FlanChanXwO/javdb-cli/pull/48))
- Add client-side collection selector fan-out and machine-readable `config get` JSON/NDJSON output while retaining the documented legacy aggregate JSON and human-readable forms. ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))

## Changed

- **Breaking:** v0.7.3 users must migrate from `javdb download NUMBER --thumbnail/--preview-image/--preview-video` to the composable `javdb assets list ... | javdb assets download ...` flow. The public SDK likewise moves from `DownloadMovieMedia` with `MovieMediaDownloadOptions` / `MovieMediaDownloadResult` to `MovieAssets` plus `DownloadMovieAsset`. ([#45](https://github.com/FlanChanXwO/javdb-cli/pull/45), [#47](https://github.com/FlanChanXwO/javdb-cli/pull/47))
- **Breaking:** lists/collections NDJSON moves from aggregate `data.lists` / `data.items` payloads to per-record `data.list` / `data.entity` envelopes with required stable IDs. Legacy human-readable output and explicit aggregate `--json` keep their existing shapes. ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))
- Document the official JavDB website and the official app GitHub Releases page in both READMEs. ([#43](https://github.com/FlanChanXwO/javdb-cli/pull/43))

## Fixed

- Prevent fuzzy or ambiguous movie-number resolution from silently selecting the first search result: surrounding whitespace is ignored, case-insensitive exact matches are accepted first, and only a unique formatting-equivalent number may fall back. Also fix `mark` / `unmark` pipeline movie-ID precedence and avoid consuming stdin during `mark` status validation. ([#44](https://github.com/FlanChanXwO/javdb-cli/pull/44))
- Validate collection selectors and `magnets --min-size` before client setup or network requests, reject non-finite, negative, and overflowing sizes, preserve authoritative movie/list IDs, and fail machine envelopes that lack a stable ID. ([#46](https://github.com/FlanChanXwO/javdb-cli/pull/46))

**Full Changelog**: [v0.7.3...v0.8.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.3...v0.8.0)
