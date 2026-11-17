# v0.7.0 — 2026-08-14

## 破坏性变更

- 多数单引用命令通过 `javdb.pipeline/v1` 接受管道批处理；非 TTY 批量输出默认使用 JSONL 信封，可按需选择 `--jsonl`、`--text` 或 `--json`。结果保持输入顺序，单项失败后继续处理，并在部分失败时以非零状态退出；生产者命令保留 stdin。 ([`57d1b5b`](https://github.com/FlanChanXwO/javdb-cli/commit/57d1b5baf80e9f684247b634039e74a7c47cd8f4), [`9cd2676`](https://github.com/FlanChanXwO/javdb-cli/commit/9cd26768f2acfd816cf6b4cc017434e98af400d9), [`da2ee6d`](https://github.com/FlanChanXwO/javdb-cli/commit/da2ee6d6b824565ca854df8cfeddeb5dfcd0769c))

- 请使用 `javdb --version` 查看构建信息与 Release 链接。旧 `version --json` 兼容命令继续供旧版更新器调用，帮助与补全中隐藏该命令。 ([`e0c5846`](https://github.com/FlanChanXwO/javdb-cli/commit/e0c5846c6aefd5766441a09531a6a978a1176aa9))

## 新增

- 新增以图搜番：支持文件、HTTP(S) URL 或二进制 stdin 中不超过 8 MiB 的 JPEG、PNG、WEBP。图片上传至 AVScan 或配置的反搜服务；结果包含相似度和帧信息，并按精确番号匹配 JavDB 详情。支持配置反搜服务与环境变量请求头；`javdb cache reverse-search` 可查看或清理按服务和图片隔离的 30 天缓存，SDK 同步提供反搜接口。 ([`51aabdc`](https://github.com/FlanChanXwO/javdb-cli/commit/51aabdc6dbd0eb87c994ae8737d28f678815273d), [`f5a5c72`](https://github.com/FlanChanXwO/javdb-cli/commit/f5a5c72046205146fb0e9632cf2ff47559ceb031), [`c324b9b`](https://github.com/FlanChanXwO/javdb-cli/commit/c324b9bffe8db6c6f9a16f61a08d7c6a7b40d1ad), [`883a066`](https://github.com/FlanChanXwO/javdb-cli/commit/883a06698bbdbb28fc34fc4be0397a6edcc2ec14), [`a35ff17`](https://github.com/FlanChanXwO/javdb-cli/commit/a35ff17493366d46320f0f54d09d422ff734690d))

## 修复

- 修复 TOP250 文本榜单与管道输出参数、本地命令的 JSON 输出、`mark` 的状态参数处理，以及管道下载的 `--id` 处理。 ([`445fca5`](https://github.com/FlanChanXwO/javdb-cli/commit/445fca5b0bc1e5af98db5560ec995afbb8979b30), [`a382a5a`](https://github.com/FlanChanXwO/javdb-cli/commit/a382a5a82b1eb63ac69e326153028e1af26140cf), [`e7a37de`](https://github.com/FlanChanXwO/javdb-cli/commit/e7a37de40b1c1690e5c4902331ac8aeb41f977e2), [`a2844e3`](https://github.com/FlanChanXwO/javdb-cli/commit/a2844e3402b3e999c17bbeaa8cfed55731d9023d))

- 修复 SOCKS4 代理支持，图片 URL 下载复用配置的代理；无效图片、反搜服务名称及损坏缓存显式报错。 ([`62ba4ab`](https://github.com/FlanChanXwO/javdb-cli/commit/62ba4ab19a9a9f9944673113d32671f8461e6cce), [`d7ccdbe`](https://github.com/FlanChanXwO/javdb-cli/commit/d7ccdbe21e4916fcf7bc2f9eecb6cb6f090e8acb), [`fb63f1d`](https://github.com/FlanChanXwO/javdb-cli/commit/fb63f1df2d940dd3a3dd67f6bc5cacd5f9362408))

## 安全

- 反搜请求头校验防止敏感请求头泄漏至非预期目标。 ([`b8df453`](https://github.com/FlanChanXwO/javdb-cli/commit/b8df4539684ee3bdb4e4b4b84cf2f65a1fb4b72c), [`65d493b`](https://github.com/FlanChanXwO/javdb-cli/commit/65d493b47d2ecda1c53a1e566370eaa5dc8915bc))

**完整变更**：[v0.6.1...v0.7.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.6.1...v0.7.0)

<!-- internal:
  [`1c2dc9d`](https://github.com/FlanChanXwO/javdb-cli/commit/1c2dc9d5d32707ce58c8a1d841be41a3a5aed4dc)
  [#24](https://github.com/FlanChanXwO/javdb-cli/pull/24)
  [`26f07f7`](https://github.com/FlanChanXwO/javdb-cli/commit/26f07f762e42d7321448e616f4b0ab2cee6a032a)
  [`b46c096`](https://github.com/FlanChanXwO/javdb-cli/commit/b46c096b1dd834bf1cfeeb65a888cf658f4c072c)
  [`d16b546`](https://github.com/FlanChanXwO/javdb-cli/commit/d16b546e61efd718ea6f247d4943aa33387fb7c7)
  [`d204300`](https://github.com/FlanChanXwO/javdb-cli/commit/d204300fb30f19aa5ada461b558151c22ebd6027)
  [`2153dec`](https://github.com/FlanChanXwO/javdb-cli/commit/2153decfb191a5ed45d801d4bc113bcc6977e302)
  [`26f1ef8`](https://github.com/FlanChanXwO/javdb-cli/commit/26f1ef84a36728bd51340943cb20782339b9c6dd)
  [`7f62f60`](https://github.com/FlanChanXwO/javdb-cli/commit/7f62f6012616d988bc36be69f6d6d5f0e73c3eb2)
  [`9fd60d9`](https://github.com/FlanChanXwO/javdb-cli/commit/9fd60d9f655c86e5fa64400a2ae342e5a449c542)
  [`b15de50`](https://github.com/FlanChanXwO/javdb-cli/commit/b15de50b99fb5eaef9e6220c10b50ac84c3aaa6e)
  [`dc5fc3c`](https://github.com/FlanChanXwO/javdb-cli/commit/dc5fc3cd974d1b978ba037910f9c4d915150c9e3)
  [`ff42865`](https://github.com/FlanChanXwO/javdb-cli/commit/ff42865cae26ca28c9ca2beaded2288f0cf530b9)
-->
