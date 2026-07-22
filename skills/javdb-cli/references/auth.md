# 认证与账号

本 CLI 的账号数据在 `~/.javdb-cli/auth.json`，权限为 `0600`。它含密码和 JWT；不要读取、
打印、上传或将其加入诊断文件。

- 用户需要登录时，可让其在私人终端执行 `javdb auth login`；缺少 `-u/-p` 时会交互提示。
- 仅在用户已明确在会话中给出凭据并要求这样做时，才运行 `javdb auth login -u USER -p PASS`。执行前说明参数会进入本次命令记录；完成后只报告结果，绝不复述密码或 token。
- `javdb auth list` 不会显示 token；只在默认账号选择或用户请求列表时执行。
- `javdb auth check --json` 是网络验证。失败时先说明错误；不要自动重新登录。
- `javdb auth use USER_ID` 与 `javdb auth remove USER_ID` 改变本地账号状态，逐次取得明确授权。

若用户要求自动续登，说明它保存并使用账户密码：只有在明确同意后才设置
`javdb config set auto_relogin true`。
