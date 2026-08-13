# v0.7.0 — 2026-08-13

## 破坏性变更

- 新增可组合的 `javdb.pipeline/v1` 协议：多数单 ref 命令接受非 TTY stdin 批处理，输入按图片 magic、JSONL 信封、纯文本顺序分类；非 TTY 输出默认每行一个 JSONL 信封；`--jsonl`、`--text` 与 `--json` 互斥；批量 `--json` 输出信封数组；消费者严格检查 kind、保持输入顺序、单项失败继续执行并最终非零。生产者命令不读 stdin。([`389c487`](https://github.com/FlanChanXwO/javdb-cli/commit/389c487)、[`b58ebce`](https://github.com/FlanChanXwO/javdb-cli/commit/b58ebce)、[`e0926f3`](https://github.com/FlanChanXwO/javdb-cli/commit/e0926f3))
- 公开版本接口切换为根 `javdb --version`（gh 风格：正式版两行且不带 `v` 前缀并附 Release URL；开发版一行），并隐藏旧 `version --json` shim（仍保留供旧版更新器调用，但不再出现在 help/completion）。([`febc1fd`](https://github.com/FlanChanXwO/javdb-cli/commit/febc1fd))

## 新增

- 新增以图搜番：接受本地路径、HTTP(S) URL 或二进制 stdin 的 JPEG/PNG/WEBP 图片（最大 8 MiB）；把原始字节上传到内置 AVScan provider 或声明式外部 HTTP source；返回规范化候选与帧；并以大小写不敏感的严格番号精确匹配把每个候选联动到完整 JavDB 详情。响应按 source 与原图 SHA-256 本地缓存 30 天，`javdb cache reverse-search` 只查看或清理该缓存。([`4e30ba7`](https://github.com/FlanChanXwO/javdb-cli/commit/4e30ba7)、[`9d25d4f`](https://github.com/FlanChanXwO/javdb-cli/commit/9d25d4f)、[`e4ba213`](https://github.com/FlanChanXwO/javdb-cli/commit/e4ba213)、[`9ab06b7`](https://github.com/FlanChanXwO/javdb-cli/commit/9ab06b7)、[`0b59fdf`](https://github.com/FlanChanXwO/javdb-cli/commit/0b59fdf))

## 变更

- Release 压缩包现在随发布提供 `release-manifest.json` 与 `release-manifest.sig`（只由受保护 release environment 中已验证的 production archives 生成）；`checksums.txt` 由清单派生。更新器校验 Ed25519 签名、仓库/tag/平台绑定与归档及解包二进制的 SHA-256，绝不执行下载的候选二进制。([`7b86f8a`](https://github.com/FlanChanXwO/javdb-cli/commit/7b86f8a)、[`7e2c061`](https://github.com/FlanChanXwO/javdb-cli/commit/7e2c061)、[`258148a`](https://github.com/FlanChanXwO/javdb-cli/commit/258148a))

**完整变更记录**：[v0.6.1...v0.7.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.6.1...v0.7.0)
