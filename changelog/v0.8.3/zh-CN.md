# v0.8.3 — 2026-10-22

## 安全

- 将源码构建、CI、原生发布归档和容器二进制共用的 Go 工具链从 `1.27.1` 升级到 `1.27.3`，纳入 10 月 8 日的标准库安全修复及 10 月 19 日的 `crypto/x509` 证书信任修复 [CVE-2026-111943 / GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809)。需安装重新构建的 CLI 或容器镜像才能获得修复；仅升级 Go 不会修复既有二进制。CLI 命令、SDK API、配置及已保存的凭据保持不变。 ([#105](https://github.com/FlanChanXwO/javdb-cli/pull/105))

**完整变更**：[v0.8.2...v0.8.3](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.2...v0.8.3)
