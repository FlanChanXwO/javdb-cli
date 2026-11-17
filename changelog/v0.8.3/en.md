# v0.8.3 — 2026-10-22

## Security

- Release binaries and source builds use Go 1.27.3, including the Go 1.27.2 security fixes and the `crypto/x509` certificate-trust fix [CVE-2026-111943 / GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809). Upgrade the CLI or container image to receive these fixes; source builds require the updated toolchain. ([#105](https://github.com/FlanChanXwO/javdb-cli/pull/105))

**Full Changelog**: [v0.8.2...v0.8.3](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.2...v0.8.3)
