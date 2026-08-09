# v0.5.1 — 2026-08-09

## Added

- Add `--json` output to the `rankings movies`, `rankings actors`, `rankings playback`, and `top250` commands, preserving `--has-magnets` filtering and stable `movies`/`actors` result keys. ([#14](https://github.com/FlanChanXwO/javdb-cli/pull/14))

## Fixed

- Fix movie and playback rankings query mapping by translating textual zones such as `censored`, `uncensored`, `western`, and `fc2` to the App API's numeric values, and normalizing ranking periods consistently across ranking endpoints. ([#13](https://github.com/FlanChanXwO/javdb-cli/pull/13))

## New Contributors

- [@kanoshiou](https://github.com/kanoshiou) made their first contribution in [#13](https://github.com/FlanChanXwO/javdb-cli/pull/13).

**Full Changelog**: [v0.5.0...v0.5.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.0...v0.5.1)
