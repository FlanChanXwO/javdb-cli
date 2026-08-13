# v0.7.0 — 2026-08-13

## Breaking changes

- Add the composable `javdb.pipeline/v1` protocol: most single-ref commands accept non-TTY stdin batches classified as image magic, JSONL envelopes, or plain text; batch output defaults to one JSONL envelope per line off-TTY; `--jsonl`, `--text` and `--json` are mutually exclusive; batch `--json` emits an array of envelopes; consumers check kinds strictly, keep input order, continue after item failures, and exit non-zero. Producers never read stdin. ([`389c487`](https://github.com/FlanChanXwO/javdb-cli/commit/389c487), [`b58ebce`](https://github.com/FlanChanXwO/javdb-cli/commit/b58ebce), [`e0926f3`](https://github.com/FlanChanXwO/javdb-cli/commit/e0926f3))
- Switch the public version interface to the root `javdb --version` (gh style: release builds print two lines without a leading `v` plus the Release URL; development builds print one line) and hide the legacy `version --json` shim from help and completion while keeping it for older updaters. ([`febc1fd`](https://github.com/FlanChanXwO/javdb-cli/commit/febc1fd))

## Added

- Add reverse image search: accept JPEG/PNG/WEBP images up to 8 MiB from local paths, HTTP(S) URLs or binary stdin; upload original bytes to the built-in AVScan provider or a declarative external HTTP source; return normalized candidates and frames; and link every candidate to full JavDB detail through strict case-insensitive exact number matching. Responses are cached locally for 30 days keyed by source and image SHA-256, and `javdb cache reverse-search` inspects or clears only that cache. ([`4e30ba7`](https://github.com/FlanChanXwO/javdb-cli/commit/4e30ba7), [`9d25d4f`](https://github.com/FlanChanXwO/javdb-cli/commit/9d25d4f), [`e4ba213`](https://github.com/FlanChanXwO/javdb-cli/commit/e4ba213), [`9ab06b7`](https://github.com/FlanChanXwO/javdb-cli/commit/9ab06b7), [`0b59fdf`](https://github.com/FlanChanXwO/javdb-cli/commit/0b59fdf))

## Changed

- Release archives are now published with a signed `release-manifest.json` and `release-manifest.sig` generated only from verified production archives in the protected release environment; `checksums.txt` is derived from the manifest. The updater verifies the Ed25519 signature, repository/tag/platform binding, and archive plus extracted binary SHA-256 without ever executing the downloaded candidate. ([`7b86f8a`](https://github.com/FlanChanXwO/javdb-cli/commit/7b86f8a), [`7e2c061`](https://github.com/FlanChanXwO/javdb-cli/commit/7e2c061), [`258148a`](https://github.com/FlanChanXwO/javdb-cli/commit/258148a))

**Full Changelog**: [v0.6.1...v0.7.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.6.1...v0.7.0)
