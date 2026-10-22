# v0.8.3 — 2026-10-22

## Security

- Upgrade the shared Go toolchain from `1.27.1` to `1.27.3` for source builds, CI, native release archives, and container binaries. This includes the October 8 standard-library security fixes and the October 19 `crypto/x509` certificate-trust fix, [CVE-2026-111943 / GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809). Install the rebuilt CLI or container image to receive these fixes; upgrading Go alone does not patch an existing binary. CLI commands, SDK APIs, configuration, and stored credentials are unchanged. ([#105](https://github.com/FlanChanXwO/javdb-cli/pull/105))

**Full Changelog**: [v0.8.2...v0.8.3](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.2...v0.8.3)
