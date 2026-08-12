# v0.6.0 — 2026-08-12

## Added

- Add automatic API host selection: reuse a valid cached route, otherwise probe startup candidates concurrently and persist the fastest host; expose explicit public SDK support and make auto the CLI default while preserving explicit hosts. Configuration creation is atomic, update proxy resolution is independent of JavDB host settings, invalid proxy ports and negative retries are rejected, and unmark sends large review IDs as exact decimals. ([#22](https://github.com/FlanChanXwO/javdb-cli/pull/22))

## Changed

- Retire the SkillHub publisher, keep ClawHub as the active Agent skill distribution path, and document public ClawHub installation with version pinning. ([#19](https://github.com/FlanChanXwO/javdb-cli/pull/19), [#20](https://github.com/FlanChanXwO/javdb-cli/pull/20))

## Maintenance

- Verify ClawHub publication through its public skill endpoint without exposing publisher credentials during moderation. ([#18](https://github.com/FlanChanXwO/javdb-cli/pull/18))
- Remove internal compatibility facades and realign the CLI around real commands; split app capabilities into invocation, client and authstore packages, and unify movie, magnet and named projections under result without changing public contracts. ([#21](https://github.com/FlanChanXwO/javdb-cli/pull/21))

**Full Changelog**: [v0.5.2...v0.6.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.2...v0.6.0)
