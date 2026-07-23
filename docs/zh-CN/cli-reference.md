# javdb CLI 参考

[文档导航](../index.md) · [English](../en/cli-reference.md)

本文是 `javdb` 二进制的公开命令契约。自动化前先运行
`javdb <command> --help`；已安装版本的 help 才是精确可用 flag 的依据。

## 全局参数与配置

所有远程命令都接受以下持久参数：

| 参数 | 说明 |
| --- | --- |
| `--proxy URL` | 仅本次调用使用的 HTTP(S) 代理。 |
| `--host mirror\|main` | 仅本次调用选择的 App API 主机。 |

配置优先级依次为命令行参数、环境变量、`~/.javdb-cli/config.toml`、内置默认值。以下
命令会修改本机配置，应明确执行：

```bash
javdb config path
javdb config get [KEY]
javdb config set KEY VALUE
javdb config unset KEY
```

支持的键为 `host`、`https_proxy`（或 `proxy`）、`auto_relogin`、`lang`。默认关闭
`auto_relogin`；显式开启后，过期 JWT 才可能使用默认账号已保存的密码重登一次。

## 登录与本机状态

```bash
javdb auth login [-u USER] [-p PASS] [--use]
javdb auth list
javdb auth use USER_ID
javdb auth remove USER_ID
javdb auth check [--json]
```

- 省略 `-u` 或 `-p` 时交互输入；TTY 密码不会回显。
- `auth login` 与 `auth check` 不会打印 JWT。
- 账号数据位于 `~/.javdb-cli/auth.json`；支持 POSIX 权限的平台使用 `0600`，Windows
  不以相同方式公开 POSIX mode bits。
- `auth use` 会修改默认账号，`auth remove` 会删除账号；两者都是明确状态变更。

不要把密码或 JWT 写入命令记录、issue、聊天或源码。磁力、TOP250 和个人列表命令需要默认账号。

## 只读发现

```bash
javdb search KEYWORD [--zone ZONE] [--sort SORT] [--filter-by FILTER] \
  [--type TYPE] [--page N] [--limit N] [--has-magnets] [--json]
javdb detail NUMBER [--id] [--magnets] [--json]
javdb tags [--zone ZONE] [--refresh]
javdb browse [--zone ZONE] [--tag REF]... [--main FLAG]... [--year YYYY] \
  [--month MONTH] [--sort SORT] [--order asc|desc] [--page N] [--limit N] [--json]
```

`search --zone` 可用 `censored`、`uncensored`、`western`、`fc2`、`all`；`--type`
可选 `movie`、`code`、`series`、`actor`、`maker`、`director`、`list`。`detail --json`
会给出可传给实体命令的图关系 ID。`tags --refresh` 会下载并改写本机公开标签缓存，故不是纯本机只读操作。

`browse --tag` 接受 tag ID、英文名或中文名；`--main` 可重复传递服务端分类掩码。
程序应使用 `--json`；制表符文本面向人阅读，不是稳定机器 schema。

## 实体与合集导航

```bash
javdb actor REF [ENTITY OPTIONS]
javdb series REF [ENTITY OPTIONS]
javdb maker REF [ENTITY OPTIONS]
javdb director REF [ENTITY OPTIONS]
javdb code REF [ENTITY OPTIONS]
javdb list REF [ENTITY OPTIONS]

javdb lists [--page N] [--limit N] [--sort-by ORDER] [--json]
javdb lists show REF [--json]
javdb lists search KEYWORD [--zone ZONE] [--page N] [--limit N] [--json]
javdb lists related NUMBER [--id] [--page N] [--limit N] [--json]
```

实体命令支持分区、可重复 tag/main、排序、分页、`--has-magnets` 和 JSON 输出。
无子命令的 `lists` 读取当前登录用户的合集；`list REF` 是公开或用户合集的实体片单命令。

## 磁力、排行与个人状态

```bash
javdb magnets NUMBER [--id] [--cnsub] [--hd] [--min-size SIZE] [--best] [--json]
javdb rankings movies [--type TYPE] [--period day|week|month] [--has-magnets]
javdb rankings actors [--period day|week|month]
javdb rankings playback [--filter-by TYPE] [--period day|week|month] [--has-magnets]
javdb top250 [--zone ZONE] [--year YYYY] [--from RANK] [--page N] [--limit N] \
  [--ignore-watched] [--has-magnets]

javdb watched [--has-magnets]
javdb want [--has-magnets]
javdb recent [--has-magnets]
javdb collections actors|series|codes|makers|directors
javdb mark NUMBER --watched|--want [--score N] [--content TEXT] [--id]
javdb unmark NUMBER [--id]
```

`magnets` 与 `top250` 需要登录。`--best` 只从服务端返回的磁力集中选择，不会下载。
`mark`/`unmark` 会改写远程的看过/想看状态；`mark` 必须且只能传入
`--watched` 或 `--want` 之一。替他人或其他账号操作前必须确认。

## 更新与版本

```bash
javdb update [--check] [--prerelease] [--json]
javdb version [--json]
```

`update` 只在用户显式调用时执行，不会后台自动更新。`update --check` 只查询 GitHub Releases，
输出 `source`、`current_version`、`latest_version`、`latest_prerelease` 与 `update_available`。
只有 `--check` 可组合 `--json`，以获得该机器可读结果；不加 `--check` 时，仅在存在更高的所选版本后安装。

命令会保留安装渠道：Homebrew 使用 Formula，`go install` 使用精确 Release tag 重新安装，Release
压缩包只下载匹配平台的资产。压缩包安装先按同一 Release 的 `checksums.txt` 校验 SHA-256，再运行下载
二进制的 `version --json`，两者通过后才替换。`--prerelease` 会纳入预发布 tag；Homebrew 安装无法
安装该类 tag。`--proxy` 用于 GitHub 请求；`--host` 不生效，因为 update 不会访问 JavDB App API。

开发构建（`version=dev`）会明确拒绝自更新，应先安装已发布版本。Windows 成功替换后会暂存旧二进制
为 `.old`，下一次启动 javdb 时自动清理。

```bash
javdb version [--json]
```

`version --json` 输出 `version`、`commit`、`build_date`。支持 `--json` 的命令会将
stdout 保留为 JSON 结果；请求失败会以非零退出码和 stderr 诊断显式呈现，不伪造为空结果。

## 安全自动化流程

1. 用 `search --json` 或 `detail --json` 获取影片或图关系 ID。
2. 下一条命令仅传入返回 ID 或用户明确选定的文本，并用 `--help` 核验 flag。
3. 使用 `magnets --best --json` 前，确认磁力 URI 的用途在用户授权范围内。
4. 登录、刷新标签、改配置、选账号、`mark`/`unmark` 都是状态变更，执行前确认。

coding agent 的确认、凭据与错误处理规则见
[javdb-cli operator skill](../../skills/javdb-cli/SKILL.md)。
