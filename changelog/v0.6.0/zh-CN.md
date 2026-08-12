# v0.6.0 — 2026-08-12

## 新增

- 新增动态选线：解密启动载荷提取候选 API 域名，以可取消的 bootstrap 并发探测主机，并把优选线路持久化到本机私有缓存。 ([`f25b838`](https://github.com/FlanChanXwO/javdb-cli/commit/f25b8387e7c51b71670294856903410e6f51e6dc), [`1361760`](https://github.com/FlanChanXwO/javdb-cli/commit/136176084e1a88a49d40befa40d6c26ab8f86c1e), [`53bed64`](https://github.com/FlanChanXwO/javdb-cli/commit/53bed64e86f3b9acb090f4f0181ff430eb7840f0))
- 公开 javdb SDK 新增显式 SelectAutoHost 能力，基于选线引擎组合而成。 ([`d144128`](https://github.com/FlanChanXwO/javdb-cli/commit/d144128a2bc183b4777f011f5052d0870a535971))
- CLI 在构造 client 前先经路由缓存完成 auto 选线，参数校验与本机状态创建始终先于可能失败的网络选线。 ([`2f117a8`](https://github.com/FlanChanXwO/javdb-cli/commit/2f117a8e41546a9b8a004d58f9ebdc8f50239042))
- transport 请求支持 context 取消与零重试控制，被取消的 bootstrap 不再被重新测量。 ([`5104a03`](https://github.com/FlanChanXwO/javdb-cli/commit/5104a03faea8daf1027cd85620b6fcb64888e2dc))

## 变更

- 默认 host 由 mirror 改为 auto；config.toml 与 device_uuid 仅在参数校验通过后、由真正执行的命令创建。 ([`3962105`](https://github.com/FlanChanXwO/javdb-cli/commit/3962105f7d66e6730917df05cee0d019a0f577d9), [`a875a81`](https://github.com/FlanChanXwO/javdb-cli/commit/a875a81b0d38ee21c0257f3dabb7166df10e7ba9))

## 修复

- 在 provision device UUID、创建配置或选线之前无副作用校验 host/proxy；修正离线认证顺序与被取消 bootstrap 的测量。 ([`71454fe`](https://github.com/FlanChanXwO/javdb-cli/commit/71454fea0f31f92ea3cf436673a3caccb77b6543), [`b60ae42`](https://github.com/FlanChanXwO/javdb-cli/commit/b60ae42368d85d36188e7a18ed238db37c26057e), [`95c4fa7`](https://github.com/FlanChanXwO/javdb-cli/commit/95c4fa7b0670d75f3abb9037c3d0c6665095ef03), [`8136f2e`](https://github.com/FlanChanXwO/javdb-cli/commit/8136f2e37ace125266051c2884b87baf886d2dc3))

## 文档

- 同步 auto 选线的双语 CLI、SDK 与维护文档。 ([`e00f9d6`](https://github.com/FlanChanXwO/javdb-cli/commit/e00f9d6d814041995107540c277f45f90bea2776))

## 维护

- 移除内部兼容 facade，并按真实命令重整 CLI：根包直接组装最终命令树，App API 改为真实 Client 组合层，config/update/release-note 根目录不再保留 alias 或 forwarder。 ([`9e8f965`](https://github.com/FlanChanXwO/javdb-cli/commit/9e8f965dce1f7153db540338849bea8a0d944463))
- 重整 CLI 内部边界：根包只保留 root.go 与 root_test.go，以 invocation/client/authstore 能力包替代 app，movie/magnet 投影统一到 result，entity 只保留共享查询用例，命令专属测试回归各命令包。 ([`9d12235`](https://github.com/FlanChanXwO/javdb-cli/commit/9d12235081241bb874ebcb4a2667c7729d29a14a))
- 移除仓库中追踪的 goal 与 ADR 产物，并用 min 内建函数简化线路解密。 ([`2d07eef`](https://github.com/FlanChanXwO/javdb-cli/commit/2d07eef27723d204b85b201d15a661d54eefa4ef), [`1a306d7`](https://github.com/FlanChanXwO/javdb-cli/commit/1a306d791a872da91f1fa4c78da95c6a80ad2396))

**完整变更**：[v0.5.2...v0.6.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.2...v0.6.0)
