# v0.6.1 — 2026-08-13

## Added

- Add the v1 signed release manifest protocol: canonical JSON encoding, strict parsing, Ed25519 key_id derivation, single-signature and rotation double-signature support, an embedded trusted public keyring, and the sign-release tool that reads private key seeds only from the `JAVDB_RELEASE_ED25519_PRIVATE_KEYS` environment variable without writing or printing secrets. ([`7b86f8a`](https://github.com/FlanChanXwO/javdb-cli/commit/7b86f8a))

## Changed

- Verify release updates with the signed manifest instead of executing the candidate binary: bind repository, release tag and platform, check the archive and the extracted binary SHA-256, and keep the current executable unchanged on any failure. The candidate binary is never executed. ([`7e2c061`](https://github.com/FlanChanXwO/javdb-cli/commit/7e2c061))
- Publish each release with `release-manifest.json` and `release-manifest.sig` generated only from the verified production archives in the protected release environment, and derive the compatible `checksums.txt` from the manifest for Homebrew, v0.6.0 updaters and manual verification. ([`7e2c061`](https://github.com/FlanChanXwO/javdb-cli/commit/7e2c061))

## Maintenance

- Document the Ed25519 release-key runbook: generation, rotation with dual signatures, revocation, and the lifecycle of the `JAVDB_RELEASE_ED25519_PRIVATE_KEYS` secret in the protected GitHub release environment. ([`7e2c061`](https://github.com/FlanChanXwO/javdb-cli/commit/7e2c061))

**Full Changelog**: [v0.6.0...v0.6.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.6.0...v0.6.1)
