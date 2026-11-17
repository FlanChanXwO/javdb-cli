# v0.8.4 — 2026-11-17

## 修复

- 搜索（包括图片搜索与集成磁力搜索）以及影片、演员、播放榜单现在会使用已保存的默认账号 token，修复登录后在部分 JavDB 线路上仍出现认证失败的问题。无 token 时保持匿名请求；图片下载和反搜服务仅接收各自的请求，不携带 JavDB token。认证错误原样返回，保留手动重新登录的处理方式，不增加自动重登或匿名重试。 ([#107](https://github.com/FlanChanXwO/javdb-cli/pull/107))

**完整变更**：[v0.8.3...v0.8.4](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.3...v0.8.4)
