# v0.6.0 — 2026-08-12

## 新增

- 新增 API 自动选线：解密启动线路数据，以可取消请求并发探测候选主机，把优选线路持久化到本机私有缓存，提供显式 public SDK 支持，并在保留显式 host 的同时将 auto 设为 CLI 默认值；校验现会无副作用拒绝空白 proxy 覆盖，unmark 也会以精确十进制发送大 review ID。 ([#22](https://github.com/FlanChanXwO/javdb-cli/pull/22))

## 变更

- 退役 SkillHub 发布器，保留 ClawHub 作为当前 Agent skill 分发路径，并记录公开 ClawHub 安装与版本固定方法。 ([#19](https://github.com/FlanChanXwO/javdb-cli/pull/19), [#20](https://github.com/FlanChanXwO/javdb-cli/pull/20))

## 修复

- 将 censored、uncensored、western、fc2 等排行榜文本分区映射为 App API 数字值，并统一各排行接口的周期归一化。 ([#13](https://github.com/FlanChanXwO/javdb-cli/pull/13))

## 维护

- 通过 ClawHub 的公开 skill endpoint 验证发布结果，避免在审核期间暴露发布凭据。 ([#18](https://github.com/FlanChanXwO/javdb-cli/pull/18))
- 移除内部兼容 facade 并按真实命令重整 CLI；将 app 能力拆分为 invocation、client、authstore 包，并把 movie、magnet 与 named 投影统一到 result，保持公开契约不变。 ([#21](https://github.com/FlanChanXwO/javdb-cli/pull/21))

## 新贡献者

- [@kanoshiou](https://github.com/kanoshiou) 在 [#13](https://github.com/FlanChanXwO/javdb-cli/pull/13) 中完成首次贡献。

**完整变更**：[v0.5.2...v0.6.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.2...v0.6.0)
