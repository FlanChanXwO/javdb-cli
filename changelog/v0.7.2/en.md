# v0.7.2 — 2026-08-18

## Breaking changes

- Remove the hidden compatibility `javdb version` subcommand and its JSON shim. Use `javdb --version` for human-readable build information; CI, Homebrew, E2E checks, documentation, and the operator skill now use the root flag. ([#33](https://github.com/FlanChanXwO/javdb-cli/pull/33))

## Added

- Add `search --magnets N`, stable magnet ranking and filtering, reverse-image magnet lookup, and fan-out pipeline records for movies and named entities. ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))

## Changed

- Make versioned bilingual changelog directories the sole release-note source, remove the old plan/prepare and PR metadata flow, and let docs-only changes skip the native smoke matrix while retaining the aggregate platform gate. ([#31](https://github.com/FlanChanXwO/javdb-cli/pull/31))
- Add automatic PR reviewer/assignee routing and path-based labels with pinned GitHub Actions. ([#33](https://github.com/FlanChanXwO/javdb-cli/pull/33))

## Fixed

- Make non-TTY producer output consumable as stable refs, preserve upstream error envelopes, avoid redundant movie detail requests, report partial magnet and output failures correctly, and align real API smoke checks with the output contract. ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))
- Make release-note audit resolve squash-merged PRs by verifying the exact merge commit instead of relying on a direct commit-to-PR lookup. ([#30](https://github.com/FlanChanXwO/javdb-cli/pull/30))
- Prevent skipped packaged-binary smoke jobs from exposing unexpanded matrix expressions, and align the bilingual issue templates. ([#32](https://github.com/FlanChanXwO/javdb-cli/pull/32))

## Maintenance

- Document the post-merge release preflight and final-main source audit required before creating an immutable release tag. ([#30](https://github.com/FlanChanXwO/javdb-cli/pull/30))

**Full Changelog**: [v0.7.1...v0.7.2](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.1...v0.7.2)
