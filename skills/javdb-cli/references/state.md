# 需要明确授权的状态变更

这些命令会影响远端或本地持久化状态，必须确认本次目标和动作；先前授权不自动延续。

| 动作 | 执行前需确认 |
| --- | --- |
| `javdb mark NUMBER --watched` | 影片、设置为已看、可选评分和评论文本 |
| `javdb mark NUMBER --want` | 影片、设置为想看、可选评分和评论文本 |
| `javdb unmark NUMBER` | 影片与将删除的标记 |
| `javdb auth use USER_ID` | 将要成为默认账号的 ID |
| `javdb auth remove USER_ID` | 将要删除的本地账号 ID |
| `javdb config set/unset KEY` | 配置键和新值/重置结果 |

`--content` 的文字会保存到服务端。遇到含敏感信息、身份信息或攻击性内容时，先请用户复核；不要从上下文自动生成并提交评论。
