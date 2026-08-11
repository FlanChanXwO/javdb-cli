# v0.5.3 — 2026-08-11

## Maintenance

- Remove the internal compatibility facades and realign the CLI around real commands: the root package now assembles the final command tree directly, App API is a real Client composition, and the config/update/release-note roots no longer keep aliases or forwarders. ([`9e8f965`](https://github.com/FlanChanXwO/javdb-cli/commit/9e8f965dce1f7153db540338849bea8a0d944463))
- Split the CLI internal boundaries: the root package keeps only root.go and root_test.go, app is replaced by invocation/client/authstore capability packages, movie/magnet projections are unified under result, entity keeps only the shared query, and command-owned tests return to their command packages. ([`9d12235`](https://github.com/FlanChanXwO/javdb-cli/commit/9d12235081241bb874ebcb4a2667c7729d29a14a))

**Full Changelog**: [v0.5.2...v0.5.3](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.2...v0.5.3)
