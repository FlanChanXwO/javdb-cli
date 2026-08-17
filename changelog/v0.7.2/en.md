# v0.7.2 — 2026-08-16

## Added

- Add `search --magnets N`, stable magnet ranking and filtering, reverse-image magnet lookup, and fan-out pipeline records for movies and named entities. ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))

## Fixed

- Make non-TTY producer output consumable as stable refs, preserve upstream error envelopes, avoid redundant movie detail requests, report partial magnet and output failures correctly, and align real API smoke checks with the output contract. ([#29](https://github.com/FlanChanXwO/javdb-cli/pull/29))

## Maintenance

- Make release-note audit resolve squash-merged PRs by verifying the exact merge commit, and document the post-merge release preflight across project skills and maintainer guidance. ([#30](https://github.com/FlanChanXwO/javdb-cli/pull/30))

**Full Changelog**: [v0.7.1...v0.7.2](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.1...v0.7.2)
