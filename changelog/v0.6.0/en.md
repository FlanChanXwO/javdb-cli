# v0.6.0 — 2026-08-12

## Added

- Add dynamic route selection: startup payloads are decrypted to extract candidate API domains, hosts are probed concurrently with cancellable bootstraps, and the preferred route is persisted in a private local cache. ([`f25b838`](https://github.com/FlanChanXwO/javdb-cli/commit/f25b8387e7c51b71670294856903410e6f51e6dc), [`1361760`](https://github.com/FlanChanXwO/javdb-cli/commit/136176084e1a88a49d40befa40d6c26ab8f86c1e), [`53bed64`](https://github.com/FlanChanXwO/javdb-cli/commit/53bed64e86f3b9acb090f4f0181ff430eb7840f0))
- Expose an explicit SelectAutoHost capability on the public javdb SDK, composed over the route engine. ([`d144128`](https://github.com/FlanChanXwO/javdb-cli/commit/d144128a2bc183b4777f011f5052d0870a535971))
- The CLI resolves the auto host through the route cache before building the client, so parameter validation and local state creation always precede fallible network selection. ([`2f117a8`](https://github.com/FlanChanXwO/javdb-cli/commit/2f117a8e41546a9b8a004d58f9ebdc8f50239042))
- Transport requests are context-aware with zero-retry control, and cancelled bootstraps are no longer re-measured. ([`5104a03`](https://github.com/FlanChanXwO/javdb-cli/commit/5104a03faea8daf1027cd85620b6fcb64888e2dc))

## Changed

- The default host is now auto instead of mirror, and config.toml plus device_uuid are created only after parameter validation by commands that actually execute. ([`3962105`](https://github.com/FlanChanXwO/javdb-cli/commit/3962105f7d66e6730917df05cee0d019a0f577d9), [`a875a81`](https://github.com/FlanChanXwO/javdb-cli/commit/a875a81b0d38ee21c0257f3dabb7166df10e7ba9))

## Fixed

- Validate host and proxy without side effects before device UUID provisioning, config creation or route selection; offline auth ordering and cancelled-bootstrap measurements are corrected. ([`71454fe`](https://github.com/FlanChanXwO/javdb-cli/commit/71454fea0f31f92ea3cf436673a3caccb77b6543), [`b60ae42`](https://github.com/FlanChanXwO/javdb-cli/commit/b60ae42368d85d36188e7a18ed238db37c26057e), [`95c4fa7`](https://github.com/FlanChanXwO/javdb-cli/commit/95c4fa7b0670d75f3abb9037c3d0c6665095ef03), [`8136f2e`](https://github.com/FlanChanXwO/javdb-cli/commit/8136f2e37ace125266051c2884b87baf886d2dc3))

## Documentation

- Sync bilingual CLI, SDK and maintainer documentation for auto host resolution. ([`e00f9d6`](https://github.com/FlanChanXwO/javdb-cli/commit/e00f9d6d814041995107540c277f45f90bea2776))

## Maintenance

- Remove the internal compatibility facades and realign the CLI around real commands: the root package now assembles the final command tree directly, App API is a real Client composition, and the config/update/release-note roots no longer keep aliases or forwarders. ([`9e8f965`](https://github.com/FlanChanXwO/javdb-cli/commit/9e8f965dce1f7153db540338849bea8a0d944463))
- Split the CLI internal boundaries: the root package keeps only root.go and root_test.go, app is replaced by invocation/client/authstore capability packages, movie/magnet projections are unified under result, entity keeps only the shared query, and command-owned tests return to their command packages. ([`9d12235`](https://github.com/FlanChanXwO/javdb-cli/commit/9d12235081241bb874ebcb4a2667c7729d29a14a))
- Remove tracked goal and ADR artifacts and simplify route decryption with the min builtin. ([`2d07eef`](https://github.com/FlanChanXwO/javdb-cli/commit/2d07eef27723d204b85b201d15a661d54eefa4ef), [`1a306d7`](https://github.com/FlanChanXwO/javdb-cli/commit/1a306d791a872da91f1fa4c78da95c6a80ad2396))

**Full Changelog**: [v0.5.2...v0.6.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.2...v0.6.0)
