# v0.8.4 — 2026-11-17

## 修复

- 搜索（包括图片与集成磁力搜索）及影片、演员、播放榜单使用已保存的默认账号 token，修复登录后在部分线路上仍认证失败的问题。无 token 时保持匿名请求；图片下载和反搜服务仅接收各自请求，不携带 JavDB token；请求被拒绝时返回认证错误。 ([#107](https://github.com/FlanChanXwO/javdb-cli/pull/107))

## 新贡献者

- 感谢 [@LightYear512](https://github.com/LightYear512) 的首次贡献。 ([#107](https://github.com/FlanChanXwO/javdb-cli/pull/107))

**完整变更**：[v0.8.3...v0.8.4](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.3...v0.8.4)
