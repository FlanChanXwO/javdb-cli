# v0.6.0 — 2026-08-12

## Added

- Add dynamic route selection: startup payloads are decrypted to extract candidate API domains, hosts are probed concurrently with cancellable bootstraps, and the preferred route is persisted in a private local cache. ([`f25b838`](https://github.com/FlanChanXwO/javdb-cli/commit/f25b8387e7c51b71670294856903410e6f51e6dc), [`1361760`](https://github.com/FlanChanXwO/javdb-cli/commit/136176084e1a88a49d40befa40d6c26ab8f86c1e), [`53bed64`](https://github.com/FlanChanXwO/javdb-cli/commit/53bed64e86f3b9acb090f4f0181ff430eb7840f0))
- Expose an explicit SelectAutoHost capability on the public javdb SDK, composed over the route engine. ([`d144128`](https://github.com/FlanChanXwO/javdb-cli/commit/d144128a2bc183b4777f011f5052d0870a535971))
- The CLI resolves the auto host through the route cache before building the client, so parameter validation and local state creation always precede fallible network selection. ([`2f117a8`](https://github.com/FlanChanXwO/javdb-cli/commit/2f117a8e41546a9b8a004d58f9ebdc8f50239042))
- Transport requests are context-aware with zero-retry control, and cancelled bootstraps are no longer re-measured. ([`5104a03`](https://github.com/FlanChanXwO/javdb-cli/commit/5104a03faea8daf1027cd85620b6fcb64888e2dc))

## Changed

- The default host is now auto instead of mirror, and config.toml plus device_uuid are created only after parameter validation by commands that actually execute. ([`3962105`](https://github.com/FlanChanXwO/javdb-cli/commit/3962105f7d66e6730917df05cee0d019a0f577d9), [`a875a81`](https://github.com/FlanChanXwO/javdb-cli/commit/a875a81b0d38ee21c0257f3dabb7166df10e7ba9))
- Retire the SkillHub publisher, keep ClawHub as the active Agent skill distribution path, and document public ClawHub installation with version pinning. ([#19](https://github.com/FlanChanXwO/javdb-cli/pull/19), [#20](https://github.com/FlanChanXwO/javdb-cli/pull/20))

## Fixed

- Validate host and proxy without side effects before device UUID provisioning, config creation or route selection; reject blank proxy overrides while accepting transport-supported proxy schemes; correct offline auth ordering and cancelled-bootstrap measurements. ([`71454fe`](https://github.com/FlanChanXwO/javdb-cli/commit/71454fea0f31f92ea3cf436673a3caccb77b6543), [`b60ae42`](https://github.com/FlanChanXwO/javdb-cli/commit/b60ae42368d85d36188e7a18ed238db37c26057e), [`95c4fa7`](https://github.com/FlanChanXwO/javdb-cli/commit/95c4fa7b0670d75f3abb9037c3d0c6665095ef03), [`8136f2e`](https://github.com/FlanChanXwO/javdb-cli/commit/8136f2e37ace125266051c2884b87baf886d2dc3), [`d94cb04`](https://github.com/FlanChanXwO/javdb-cli/commit/d94cb04303b03e203406f670f4efa55e989f64e0))
- Map textual rankings zones such as censored, uncensored, western and fc2 to App API numeric values, and normalize ranking periods consistently across endpoints. ([#13](https://github.com/FlanChanXwO/javdb-cli/pull/13))
- Format review identifiers as exact decimal values when removing watched or wanted state, avoiding scientific notation for large IDs. ([`9273cbe`](https://github.com/FlanChanXwO/javdb-cli/commit/9273cbe5e9f453af758b0c0d9b4c5522b33f8a1c))

## Documentation

- Sync bilingual CLI, SDK and maintainer documentation for auto host resolution, and document supported proxy schemes plus blank proxy-flag rejection in the operator skill. ([`e00f9d6`](https://github.com/FlanChanXwO/javdb-cli/commit/e00f9d6d814041995107540c277f45f90bea2776), [`00a4dd4`](https://github.com/FlanChanXwO/javdb-cli/commit/00a4dd4dc6d5f572f4f7052c2a908339218f3525))

## Maintenance

- Remove internal compatibility facades and realign the CLI around real commands; split app capabilities into invocation, client and authstore packages, and unify movie, magnet and named projections under result without changing public contracts. ([#21](https://github.com/FlanChanXwO/javdb-cli/pull/21))
- Verify ClawHub publication through its public skill endpoint without exposing publisher credentials during moderation. ([#18](https://github.com/FlanChanXwO/javdb-cli/pull/18))
- Remove tracked goal and ADR artifacts, simplify route decryption with the min builtin, and clean up the route selector test helper. ([`2d07eef`](https://github.com/FlanChanXwO/javdb-cli/commit/2d07eef27723d204b85b201d15a661d54eefa4ef), [`1a306d7`](https://github.com/FlanChanXwO/javdb-cli/commit/1a306d791a872da91f1fa4c78da95c6a80ad2396), [`fe52972`](https://github.com/FlanChanXwO/javdb-cli/commit/fe529729eee6e6fdd2b29194601e8c91333b0600))

## New Contributors

- [@kanoshiou](https://github.com/kanoshiou) made their first contribution in [#13](https://github.com/FlanChanXwO/javdb-cli/pull/13).

**Full Changelog**: [v0.5.2...v0.6.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.2...v0.6.0)
