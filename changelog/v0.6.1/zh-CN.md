# v0.6.1 — 2026-08-13

## 安全

- `javdb update` 在替换前验证发布清单签名、仓库、tag、平台、归档与解包二进制，全程无需执行候选程序；校验失败时保留已安装程序。发布包新增 `release-manifest.json` 与 `release-manifest.sig`，并保留供旧版更新器与 Homebrew 使用的 `checksums.txt`。 ([`7b86f8a`](https://github.com/FlanChanXwO/javdb-cli/commit/7b86f8adf63a9a2bb30f2e603fba4a48c8e27c6f), [`7e2c061`](https://github.com/FlanChanXwO/javdb-cli/commit/7e2c06165c38a092e085e7fb5492cc1b05d9e7f5), [`258148a`](https://github.com/FlanChanXwO/javdb-cli/commit/258148a6a1076867baa5b49286c983ddd0f52329), [`c7f1c97`](https://github.com/FlanChanXwO/javdb-cli/commit/c7f1c97413b49e5102d976bc29586fad052c1480))

**完整变更记录**：[v0.6.0...v0.6.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.6.0...v0.6.1)

<!-- internal:
  [`895fef2`](https://github.com/FlanChanXwO/javdb-cli/commit/895fef26867709bdf3e48b2e0719ef35d671188a)
-->
