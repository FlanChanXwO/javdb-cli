# v0.7.3 — 2026-09-08

## Added

- Publish verified Linux multi-architecture container images to GHCR and Docker Hub for stable releases, covering `linux/amd64` and `linux/arm64` with exact-version, per-architecture, and `latest` tags; Docker Hub publication starts with this release. ([#39](https://github.com/FlanChanXwO/javdb-cli/pull/39), [#40](https://github.com/FlanChanXwO/javdb-cli/pull/40))

## Changed

- Rebuild and smoke-validate container images from immutable release tags, use one protected publish job for both registries and the GitHub Release, and verify Docker Hub visibility before making the Release public. ([#39](https://github.com/FlanChanXwO/javdb-cli/pull/39), [#40](https://github.com/FlanChanXwO/javdb-cli/pull/40))
- Replace the pull request template with bilingual change, verification, checklist, and credential-safety guidance. ([#36](https://github.com/FlanChanXwO/javdb-cli/pull/36))

**Full Changelog**: [v0.7.2...v0.7.3](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.2...v0.7.3)
