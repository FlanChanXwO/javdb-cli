# v0.7.3 — 2026-09-08

## 新增

- 为 stable release 向 GHCR 与 Docker Hub 发布经过验证的 Linux 多架构容器镜像，覆盖 `linux/amd64` 与 `linux/arm64`，并提供精确版本、分架构和 `latest` tag；Docker Hub 从本版本开始发布。 ([#39](https://github.com/FlanChanXwO/javdb-cli/pull/39), [#40](https://github.com/FlanChanXwO/javdb-cli/pull/40))

## 变更

- 从不可变 release tag 重建并执行容器冒烟验证；通过一个受保护的 publish job 同时管理两个 registry 与 GitHub Release，并在公开 Release 前验证 Docker Hub 的可见性。 ([#39](https://github.com/FlanChanXwO/javdb-cli/pull/39), [#40](https://github.com/FlanChanXwO/javdb-cli/pull/40))
- 用双语 PR 模板替换旧模板，补充变更说明、验证步骤、检查清单和凭据安全要求。 ([#36](https://github.com/FlanChanXwO/javdb-cli/pull/36))

**完整变更**：[v0.7.2...v0.7.3](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.2...v0.7.3)
