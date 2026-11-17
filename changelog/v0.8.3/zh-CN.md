# v0.8.3 — 2026-10-22

## 安全

- 发布包与源码构建使用 Go 1.27.3，包含 Go 1.27.2 安全修复及 `crypto/x509` 证书信任修复 [CVE-2026-111943 / GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809)。请升级 CLI 或容器镜像以获得修复；源码构建需要更新工具链。 ([#105](https://github.com/FlanChanXwO/javdb-cli/pull/105))

**完整变更**：[v0.8.2...v0.8.3](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.2...v0.8.3)
