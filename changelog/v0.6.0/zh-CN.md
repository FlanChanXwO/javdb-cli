# v0.6.0 — 2026-08-12

## 新增

- 新增 API 自动选线：复用有效缓存线路，或选择最快的可用主机。CLI 默认使用 `auto`，仍支持显式主机；SDK 同步提供自动选线能力。 ([#22](https://github.com/FlanChanXwO/javdb-cli/pull/22))

## 变更

- Agent skill 分发渠道由 SkillHub 迁至 ClawHub，安装时可固定已发布版本。 ([#19](https://github.com/FlanChanXwO/javdb-cli/pull/19), [#20](https://github.com/FlanChanXwO/javdb-cli/pull/20))

## 修复

- `update` 独立于 JavDB API 主机解析代理。非法代理端口和负重试次数显式报错；`unmark` 精确保留大数值 review ID。 ([#22](https://github.com/FlanChanXwO/javdb-cli/pull/22))

**完整变更**：[v0.5.2...v0.6.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.2...v0.6.0)

<!-- internal:
  [#18](https://github.com/FlanChanXwO/javdb-cli/pull/18)
  [#21](https://github.com/FlanChanXwO/javdb-cli/pull/21)
-->
