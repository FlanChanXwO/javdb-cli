# v0.6.1 — 2026-08-13

## Security

- `javdb update` verifies the signed release manifest, repository, tag, platform, archive, and extracted binary before replacement, without running the candidate. Failed verification preserves the installed executable. Releases include `release-manifest.json` and `release-manifest.sig`; `checksums.txt` remains available for older updaters and Homebrew. ([`7b86f8a`](https://github.com/FlanChanXwO/javdb-cli/commit/7b86f8adf63a9a2bb30f2e603fba4a48c8e27c6f), [`7e2c061`](https://github.com/FlanChanXwO/javdb-cli/commit/7e2c06165c38a092e085e7fb5492cc1b05d9e7f5), [`258148a`](https://github.com/FlanChanXwO/javdb-cli/commit/258148a6a1076867baa5b49286c983ddd0f52329), [`c7f1c97`](https://github.com/FlanChanXwO/javdb-cli/commit/c7f1c97413b49e5102d976bc29586fad052c1480))

**Full Changelog**: [v0.6.0...v0.6.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.6.0...v0.6.1)

<!-- internal:
  [`895fef2`](https://github.com/FlanChanXwO/javdb-cli/commit/895fef26867709bdf3e48b2e0719ef35d671188a)
-->
