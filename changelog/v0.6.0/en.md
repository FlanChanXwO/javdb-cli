# v0.6.0 — 2026-08-12

## Added

- API host selection is automatic: reuse a working cached route or choose the fastest available host. `auto` becomes the CLI default; explicit hosts remain supported, with automatic selection also available through the SDK. ([#22](https://github.com/FlanChanXwO/javdb-cli/pull/22))

## Changed

- ClawHub replaces SkillHub as the Agent skill distribution channel; pin installations to a published version. ([#19](https://github.com/FlanChanXwO/javdb-cli/pull/19), [#20](https://github.com/FlanChanXwO/javdb-cli/pull/20))

## Fixed

- `update` resolves its proxy independently of the JavDB API host. Invalid proxy ports and negative retry counts fail explicitly; `unmark` preserves large review IDs exactly. ([#22](https://github.com/FlanChanXwO/javdb-cli/pull/22))

**Full Changelog**: [v0.5.2...v0.6.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.2...v0.6.0)

<!-- internal:
  [#18](https://github.com/FlanChanXwO/javdb-cli/pull/18)
  [#21](https://github.com/FlanChanXwO/javdb-cli/pull/21)
-->
