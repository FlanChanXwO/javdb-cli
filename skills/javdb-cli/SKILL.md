---
name: javdb-cli
description: 通过 javdb-cli 的 `javdb` 二进制检索 JavDB App API 的影片、人物、标签、榜单与合集，并在用户明确授权时管理账号、配置和观看标记。仅当用户明确提到 JavDB、javdb-cli、`javdb` 命令，或提供明显的 JavDB 编号/链接并要求操作时加载；不要用于泛搜索、泛成人内容或下载请求。每次执行前以 `javdb <command> --help` 核对当前可用参数。
---

# javdb-cli Operator

本 skill 让 agent 安全、准确地操作 `javdb`。命令语义以当前安装二进制的
`javdb <command> --help` 为准；本文件只给出流程、安全边界与易混淆语义。

## 预检与账号检查

1. 运行 `javdb version --json`。这是唯一默认环境检查；二进制不存在或不可执行时，说明阻塞原因。只有用户明确要求安装或修复时，才阅读 `references/install.md`，不得猜测安装方式。
2. 不要在每轮会话枚举本地账号。只有认证、默认账号选择或用户明确要求时才运行 `javdb auth list`；账号存在不代表 token 有效。
3. 仅在确实需要联网验证身份时运行 `javdb auth check --json`。它不会打印 token，但会发出 API 请求。

## 不可违反的规则

1. `~/.javdb-cli/auth.json` 含用户名、密码与 JWT（支持 POSIX 权限的平台使用 `0600`）。绝不在 commentary、结果、日志或代码块中回显、总结或转述这些值，也不读取该文件内容来“帮助排错”。
2. 交互式 `javdb auth login` 仅在用户明确要求且能亲自输入凭据时运行。没有可交互终端时，不启动一个会永久等待的登录进程。
3. 用户已经明确提供用户名和密码并明确要求登录时，才可使用 `auth login -u … -p …`；执行前说明凭据会进入本次命令的进程参数与记录。不要复述密码。
4. 账号切换/删除、配置写入、观看标记和取消标记都是状态变更。必须有本次操作的明确用户指令；授权不自动延续至后续目标。
5. 不接受 Cookie、浏览器会话或其他网页抓取替代方案。此 CLI 使用 App API；认证或 API 错误应原样说明，不要暗中换数据源、重试或伪造空结果。
6. `auto_relogin=true` 会在默认账号 JWT 失效时，使用已保存密码静默重登**一次**。这是持久化配置，只有用户明确要求才可通过 `javdb config set auto_relogin true` 开启。

## 操作分级

| 类型 | 命令 | Agent 行为 |
| --- | --- | --- |
| 只读 | `search`、`detail`、实体命令、`rankings`、`tags`、`browse`、`version`、`config get/path`、`update --check` | 用户任务需要时执行 |
| 软件更新 | `update` | 仅在用户明确要求升级时执行；会联网并替换二进制，或调用 Homebrew／`go install` |
| 认证诊断 | `auth list/check` | 仅在账号选择或认证判定需要时执行 |
| 远端状态写入 | `mark`、`unmark` | 执行前说明影片与将要设置/删除的状态 |
| 本地账号/配置写入 | `auth login/use/remove`、`config set/unset` | 每次都需要明确授权 |
| 本地缓存写入 | `tags --refresh` | 仅在用户要求刷新标签或允许重建缓存时执行 |

`magnets`、`top250`、`watched`、`want`、`recent`、`collections` 与默认 `lists`
需要默认登录账号。`detail --magnets` 同样需要认证。

## 输出与参数控制

1. 先按用户的范围表达请求；仅在该命令 `--help` 显示 `--limit` 且用户给出条数时传入正数。不要为节省上下文擅自附加限额、页数、超时或重试次数。
2. 小结果供人阅读时使用默认制表符文本；需要提取 ID、过滤、串联后续命令或需要稳定字段时使用 `--json`。
3. `--json` 只描述成功输出。先检查命令退出状态；遇到认证、参数、网络或服务端错误时，报告 stderr 的真实原因，不要把它解析成 JSON 或伪装为“无结果”。
4. `--all` 只在用户明确要求完整遍历时使用；它仅出现在实体/合集电影列表等支持的命令上。不要把它加到不支持的命令，也不要猜测 CLI 内部的分页行为。
5. `--best` 会把 `magnets` 的结果缩为单个优先项（中字 > HD > 体积）。用户要完整列表时不要添加它。

## 命令速查

执行前仍需用 `--help` 核对标志；以下示例是导航，不是稳定 API 合约。

```text
javdb version --json
javdb update --check --json
javdb config path
javdb config get host
javdb auth list
javdb auth check --json

javdb search "SSIS-589" --limit 5 --json
javdb search "巨乳" --type actor --json
javdb detail SSIS-589 --json
javdb detail MOVIE_ID --id --json        # 仅当 MOVIE_ID 已确认是内部 ID
javdb magnets SSIS-589 --cnsub --hd --json
javdb magnets SSIS-589 --best --json

javdb tags --zone censored
javdb browse --tag 巨乳 --main m --limit 20 --json
javdb actor "山手梨愛" --main m --has-magnets --json
javdb series SERIES_ID --page 1 --limit 20 --json
javdb rankings movies --period week
javdb top250 --limit 20

javdb lists search "关键词" --zone all --json
javdb list LIST_ID --json
javdb lists related SSIS-589 --json

javdb mark SSIS-589 --want
javdb unmark SSIS-589
```

所有数据命令可加的全局参数只有本次调用生效：`--proxy URL` 与
`--host mirror|main`。实现也接受用户明确批准的绝对 API URL；不要把未审阅的
URL 写入持久化配置。配置优先级为 CLI 参数 > 环境变量 > `config.toml` > 默认值。

## 关键语义与常见陷阱

1. `detail NUMBER` 默认把参数作为番号解析；`--id` 表示内部 movie ID。没有可靠来源时不要猜测并加 `--id`。
2. `list REF` 是某个公开/用户合集中的电影；不带子命令的 `lists` 是“我的合集”，需要认证。`lists show/search/related` 与 `list` 的含义不同。
3. `search --type` 可返回 `movie` 以外的维度。将搜索结果交给 `actor`、`series`、`maker`、`director`、`code` 或 `list` 前，使用 JSON 中的实际 ID/名称，不要从显示文本臆测。
4. `tags` 的首次调用可能联网建立缓存；`--refresh` 会明确覆写该缓存。标签参数可用 ID、英文名或中文名，优先使用刚读取到的确切值。
5. `mark` 必须在 `--watched` 与 `--want` 中二选一；`--content` 是要保存到远端的文本，提交前应让用户确认其内容与目标。
6. `auth check`、磁力、TOP250 和用户列表的失败是认证或网络问题的信号，不应自动登录、重设账号或切换 `host`。只有用户明确要求时才改变配置或账号。
7. `update --check --json` 是唯一可机器读取且不改写安装的更新方式。`update` 会按已检测的 Homebrew、`go install` 或 Release 压缩包渠道安装；开发构建会拒绝自更新。预发布版本只能在用户明确要求时加 `--prerelease`，且 Homebrew 渠道不支持它。

## 路由

| 任务 | 读取 |
| --- | --- |
| 明确安装或修复 `javdb` | `references/install.md` |
| 登录、账号检查、token 失效 | `references/auth.md` |
| 搜索、详情、实体图导航、合集 | `references/discover.md` |
| 标记、账号或配置变更 | `references/state.md` |
| 网络、代理、空结果或命令失败 | `references/troubleshooting.md` |
