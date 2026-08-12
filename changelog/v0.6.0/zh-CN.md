# v0.6.0 — 2026-08-12

## 新增

- 新增 API 自动选线：缓存线路有效时直接复用，否则并发探测 startup 候选并持久化最快主机；提供显式 public SDK 支持，并在保留显式 host 的同时将 auto 设为 CLI 默认值。配置文件以原子方式创建，update 代理解析不再依赖 JavDB host 设置，非法 proxy 端口和负重试次数会被拒绝，unmark 也会以精确十进制发送大 review ID。 ([#22](https://github.com/FlanChanXwO/javdb-cli/pull/22))

## 变更

- 退役 SkillHub 发布器，保留 ClawHub 作为当前 Agent skill 分发路径，并记录公开 ClawHub 安装与版本固定方法。 ([#19](https://github.com/FlanChanXwO/javdb-cli/pull/19), [#20](https://github.com/FlanChanXwO/javdb-cli/pull/20))

## 维护

- 通过 ClawHub 的公开 skill endpoint 验证发布结果，避免在审核期间暴露发布凭据。 ([#18](https://github.com/FlanChanXwO/javdb-cli/pull/18))
- 移除内部兼容 facade 并按真实命令重整 CLI；将 app 能力拆分为 invocation、client、authstore 包，并把 movie、magnet 与 named 投影统一到 result，保持公开契约不变。 ([#21](https://github.com/FlanChanXwO/javdb-cli/pull/21))

**完整变更**：[v0.5.2...v0.6.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.2...v0.6.0)
