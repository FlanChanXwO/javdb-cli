# v0.6.0 — 2026-08-12

## 新增

- 新增动态选线：解密启动载荷提取候选 API 域名，以可取消的 bootstrap 并发探测主机，并把优选线路持久化到本机私有缓存。 ([`f25b838`](https://github.com/FlanChanXwO/javdb-cli/commit/f25b8387e7c51b71670294856903410e6f51e6dc), [`1361760`](https://github.com/FlanChanXwO/javdb-cli/commit/136176084e1a88a49d40befa40d6c26ab8f86c1e), [`53bed64`](https://github.com/FlanChanXwO/javdb-cli/commit/53bed64e86f3b9acb090f4f0181ff430eb7840f0))
- 公开 javdb SDK 新增显式 SelectAutoHost 能力，基于选线引擎组合而成。 ([`d144128`](https://github.com/FlanChanXwO/javdb-cli/commit/d144128a2bc183b4777f011f5052d0870a535971))
- CLI 在构造 client 前先经路由缓存完成 auto 选线，参数校验与本机状态创建始终先于可能失败的网络选线。 ([`2f117a8`](https://github.com/FlanChanXwO/javdb-cli/commit/2f117a8e41546a9b8a004d58f9ebdc8f50239042))
- transport 请求支持 context 取消与零重试控制，被取消的 bootstrap 不再被重新测量。 ([`5104a03`](https://github.com/FlanChanXwO/javdb-cli/commit/5104a03faea8daf1027cd85620b6fcb64888e2dc))

## 变更

- 默认 host 由 mirror 改为 auto；config.toml 与 device_uuid 仅在参数校验通过后、由真正执行的命令创建。 ([`3962105`](https://github.com/FlanChanXwO/javdb-cli/commit/3962105f7d66e6730917df05cee0d019a0f577d9), [`a875a81`](https://github.com/FlanChanXwO/javdb-cli/commit/a875a81b0d38ee21c0257f3dabb7166df10e7ba9))
- 退役 SkillHub 发布器，保留 ClawHub 作为当前 Agent skill 分发路径，并记录公开 ClawHub 安装与版本固定方法。 ([#19](https://github.com/FlanChanXwO/javdb-cli/pull/19), [#20](https://github.com/FlanChanXwO/javdb-cli/pull/20))

## 修复

- 在 provision device UUID、创建配置或选线之前无副作用校验 host/proxy；拒绝空白 proxy 覆盖并接受 transport 支持的代理协议；修正离线认证顺序与被取消 bootstrap 的测量。 ([`71454fe`](https://github.com/FlanChanXwO/javdb-cli/commit/71454fea0f31f92ea3cf436673a3caccb77b6543), [`b60ae42`](https://github.com/FlanChanXwO/javdb-cli/commit/b60ae42368d85d36188e7a18ed238db37c26057e), [`95c4fa7`](https://github.com/FlanChanXwO/javdb-cli/commit/95c4fa7b0670d75f3abb9037c3d0c6665095ef03), [`8136f2e`](https://github.com/FlanChanXwO/javdb-cli/commit/8136f2e37ace125266051c2884b87baf886d2dc3), [`d94cb04`](https://github.com/FlanChanXwO/javdb-cli/commit/d94cb04303b03e203406f670f4efa55e989f64e0))
- 将 censored、uncensored、western、fc2 等排行榜文本分区映射为 App API 数字值，并统一各排行接口的周期归一化。 ([#13](https://github.com/FlanChanXwO/javdb-cli/pull/13))
- 取消 watched 或 wanted 状态时把 review 标识格式化为精确十进制值，避免大 ID 使用科学计数法。 ([`9273cbe`](https://github.com/FlanChanXwO/javdb-cli/commit/9273cbe5e9f453af758b0c0d9b4c5522b33f8a1c))

## 文档

- 同步 auto 选线的双语 CLI、SDK 与维护文档，并在 operator skill 中记录支持的代理协议与空白 proxy flag 拒绝行为。 ([`e00f9d6`](https://github.com/FlanChanXwO/javdb-cli/commit/e00f9d6d814041995107540c277f45f90bea2776), [`00a4dd4`](https://github.com/FlanChanXwO/javdb-cli/commit/00a4dd4dc6d5f572f4f7052c2a908339218f3525))

## 维护

- 移除内部兼容 facade 并按真实命令重整 CLI；将 app 能力拆分为 invocation、client、authstore 包，并把 movie、magnet 与 named 投影统一到 result，保持公开契约不变。 ([#21](https://github.com/FlanChanXwO/javdb-cli/pull/21))
- 通过 ClawHub 的公开 skill endpoint 验证发布结果，避免在审核期间暴露发布凭据。 ([#18](https://github.com/FlanChanXwO/javdb-cli/pull/18))
- 移除仓库中追踪的 goal 与 ADR 产物，用 min 内建函数简化线路解密，并清理 route selector 测试辅助逻辑。 ([`2d07eef`](https://github.com/FlanChanXwO/javdb-cli/commit/2d07eef27723d204b85b201d15a661d54eefa4ef), [`1a306d7`](https://github.com/FlanChanXwO/javdb-cli/commit/1a306d791a872da91f1fa4c78da95c6a80ad2396), [`fe52972`](https://github.com/FlanChanXwO/javdb-cli/commit/fe529729eee6e6fdd2b29194601e8c91333b0600))

## 新贡献者

- [@kanoshiou](https://github.com/kanoshiou) 在 [#13](https://github.com/FlanChanXwO/javdb-cli/pull/13) 中完成首次贡献。

**完整变更**：[v0.5.2...v0.6.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.2...v0.6.0)
