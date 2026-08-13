# v0.6.1 — 2026-08-13

## 新增

- 新增 v1 签名发布清单协议：规范 JSON 编码、严格解析、Ed25519 key_id 派生、单签与轮换双签支持、内置受信公钥环，以及只从 `JAVDB_RELEASE_ED25519_PRIVATE_KEYS` 环境变量读取私钥 seed、不写盘不打印 secret 的 sign-release 签名工具。([`7b86f8a`](https://github.com/FlanChanXwO/javdb-cli/commit/7b86f8a))

## 变更

- 发布更新改用签名清单验证，不再执行候选二进制：绑定仓库、Release tag 与平台，校验归档与解包二进制的 SHA-256，任何失败都保持现有可执行文件不变；候选二进制绝不执行。([`7e2c061`](https://github.com/FlanChanXwO/javdb-cli/commit/7e2c061))
- 每次发布新增由已验证 production archives 在受保护的 release environment 中生成的 `release-manifest.json` 与 `release-manifest.sig`，并由清单派生兼容的 `checksums.txt` 供 Homebrew、v0.6.0 更新器与人工校验使用。([`7e2c061`](https://github.com/FlanChanXwO/javdb-cli/commit/7e2c061))

## 维护

- 补充 Ed25519 发布密钥 runbook：生成、双签轮换、撤销，以及 `JAVDB_RELEASE_ED25519_PRIVATE_KEYS` 在受保护 GitHub release environment 中的生命周期说明。([`7e2c061`](https://github.com/FlanChanXwO/javdb-cli/commit/7e2c061))
- 发布审计把直接 commit 记录在报告中供人工核对，不再硬性失败，使未经过 PR 的 bridge 发布仍可发布。([`895fef2`](https://github.com/FlanChanXwO/javdb-cli/commit/895fef2))

**完整变更记录**：[v0.6.0...v0.6.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.6.0...v0.6.1)
