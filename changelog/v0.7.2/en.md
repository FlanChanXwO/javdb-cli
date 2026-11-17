# v0.7.2 — 2026-08-18

## Breaking changes

- The hidden `javdb version` compatibility command and its JSON output are removed. Use `javdb --version` for build information. ([#33](https://github.com/FlanChanXwO/javdb-cli/pull/33))

## Added

- `search --magnets N` combines movie search with magnet ranking and filtering, including reverse-image magnet lookup. Pipelines can fan out movie and named-entity records. ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))

## Fixed

- Producer text output uses stable references for downstream commands. Upstream, partial-magnet, and output errors remain visible, and redundant movie-detail requests are avoided. ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))

**Full Changelog**: [v0.7.1...v0.7.2](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.1...v0.7.2)

<!-- internal:
  [#31](https://github.com/FlanChanXwO/javdb-cli/pull/31)
  [#30](https://github.com/FlanChanXwO/javdb-cli/pull/30)
  [#32](https://github.com/FlanChanXwO/javdb-cli/pull/32)
-->
