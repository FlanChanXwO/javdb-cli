# javdb CLI 参考

[文档导航](../index.md) · [English](../en/cli-reference.md)

本文是 `javdb` 二进制的公开命令契约。自动化前先运行
`javdb <command> --help`；已安装版本的 help 才是精确可用 flag 的依据。

## 全局参数与配置

所有远程命令都接受以下持久参数：

| 参数 | 说明 |
| --- | --- |
| `--proxy URL` | 仅本次调用使用的 HTTP(S) 代理。 |
| `--host auto\|mirror\|main\|URL` | 仅本次调用选择的 App API 主机；`auto`（默认）立即复用验证成功的缓存线路，仅在验证失败时从 startup 候选中重选最快主机。 |

配置优先级依次为命令行参数、环境变量、`~/.javdb-cli/config.toml`、内置默认值。以下
命令会修改本机配置，应明确执行：

```bash
javdb config path
javdb config get [KEY]
javdb config set KEY VALUE
javdb config unset KEY
```

支持的键为 `host`、`https_proxy`（或 `proxy`）、`auto_relogin`、`lang`，以及
`reverse_search` 标量（`reverse_search.default_source`、
`reverse_search.cache`、`reverse_search.cache_ttl`、`reverse_search.retries`、
`reverse_search.retry_wait`、`reverse_search.request_timeout`）。默认关闭
`auto_relogin`；显式开启后，过期 JWT 才可能使用默认账号已保存的密码重登一次。

`config get` 无 key 时在 TTY 打印常用键，非 TTY 从 stdin 读取 key 批处理；
`config get`/`config unset` 也接受管道传入的 `config_key` 信封或纯文本 key 行。
`config set` 始终使用两个显式参数，绝不读 stdin。反搜 source 由用户在 TOML
`[[reverse_search.sources]]` 手工编辑，见「以图搜番」。

全新机器上第一个真实命令会创建 `~/.javdb-cli/config.toml`，只包含上述常用键；help、
裸/父命令、`version`、completion、参数校验失败以及缺失配置上的 `config unset` 都不会创建
或覆盖它。默认 `host` 为 `auto`：每个真实命令前 CLI 先用一次签名 `/startup` 请求验证缓存
线路，仅在验证失败时才重跑全量发现，并把新主机持久化到 `~/.javdb-cli/route.json`（mode
`0600`，只保存已验证的 host URL）。损坏缓存、重选失败或写入失败都是显式错误，绝不静默回退。
固定为 `mirror`、`main` 或绝对 URL 即可完全绕过线路发现。

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

不要把密码或 JWT 写入命令记录、issue、聊天或源码。磁力命令无需登录即可使用（有默认账号
token 时自动携带，失效则回退匿名）；TOP250 和个人列表命令需要默认账号。

## 只读发现

```bash
javdb search KEYWORD|IMAGE [--zone ZONE] [--sort SORT] [--filter-by FILTER] \
  [--type TYPE] [--page N] [--limit N] [--has-magnets] \
  [--image] [--source NAME] [--no-cache] [--json|--ndjson]
javdb detail NUMBER [--id] [--magnets] [--json|--ndjson]
javdb comments NUMBER [--id] [--page N] [--limit N] [--json|--ndjson]
javdb tags [--zone ZONE] [--refresh] [--json|--ndjson]
javdb browse [--zone ZONE] [--tag REF]... [--main FLAG]... [--year YYYY] \
  [--month MONTH] [--sort SORT] [--order asc|desc] [--page N] [--limit N] [--json|--ndjson]
```

`search --zone` 可用 `censored`、`uncensored`、`western`、`fc2`、`all`；`--type`
可选 `movie`、`code`、`series`、`actor`、`maker`、`director`、`list`。`detail --json`
会给出可传给实体命令的图关系 ID。`tags --refresh` 会下载并改写本机公开标签缓存，故不是纯本机只读操作。

`browse --tag` 接受 tag ID、英文名或中文名；`--main` 可重复传递服务端分类掩码。
程序应使用 `--json`；制表符文本面向人阅读，不是稳定机器 schema。

`comments` 对所选页只调用一次影片评论接口，不会预取或自动跟随下一页。默认读取第 `1` 页、
每页 `20` 条；需要其他**单页**时传入任意正数 `--page` 与 `--limit`。`--json` 会保留该页
上游返回的完整评论对象。

## 本地媒体下载

```bash
javdb download NUMBER [--id] [--thumbnail PATH] [--preview-image PATH] [--preview-video PATH]
```

至少设置一个输出 flag。`--thumbnail` 写入详情缩略图；`--preview-image` 只写入
`preview_images[0]`（首张预览图），不会枚举或自动改取后续预览图。`--preview-video` 将完整
HLS 预览流写入指定路径，playlist 使用 AES-128 时会解密。当前预览流是 transport stream，建议
目标路径使用 `.ts` 后缀。

命令只创建新文件：目标已存在会明确失败，也不会创建缺失的父目录。它接受已结束的单媒体 HLS
playlist；master playlist、byte-range 媒体、fragmented MP4 媒体，以及未结束/直播 playlist 都会
明确失败，不会写出不完整文件。

## 以图搜番

```bash
javdb search IMAGE|URL|--image [--source NAME] [--no-cache] [--json|--ndjson]
javdb cache reverse-search [--source NAME] [--clear]
```

`javdb search` 接受本地 JPEG/PNG/WEBP 图片（最大 8 MiB）、HTTP(S) 图片 URL 或 stdin
的二进制图片字节；`--image` 强制图片模式，现有文件或 HTTP(S) URL 参数自动识别。图片按原样上传
到已配置的 source：内置 AVScan provider（`https://avscan.cc/search`）或声明式外部 source，
最多三次总请求，对 HTTP 429、单次超时与临时传输错误按 30s/60s 退避。每个候选以大小写不敏感的
严格番号精确匹配联动 JavDB 并返回完整详情（不回退首项）；候选部分失败会完成全部输出后以非零退出。

配置位于 `config.toml`：

```toml
[reverse_search]
default_source = "builtin"
cache = true
cache_ttl = "720h"
retries = 3
retry_wait = "30s"
request_timeout = "60s"

[[reverse_search.sources]]
name = "custom"
url = "https://example.test/search"

[reverse_search.sources.headers]
Authorization = "Bearer ${ENV:REVERSE_SEARCH_TOKEN}"
```

header 值只支持静态文本加 `${ENV:NAME}` 引用；缺失变量只按名字报告，绝不携带值。source 名
唯一且只允许字母、数字、`-` 与 `_`；`builtin` 为保留名。响应缓存于
`~/.javdb-cli/reverse-search-cache`（`0600`，键为 source + 原图 SHA-256，TTL 30 天）；
缓存不保存原图、鉴权 header 或 JavDB 详情。`javdb cache reverse-search --clear [--source NAME]`
只清理反搜缓存。

隐私：反搜会把你的图片上传到已配置的 provider（默认内置 AVScan）。图片 URL 允许指向私网；
SDK 嵌入方必须自行施加网络边界。

## 管道协议

接受单个位置 ref 的命令也接受非 TTY stdin 批处理。输入按固定顺序分类：图片 magic、
`javdb.pipeline/v1` NDJSON 信封、逐行纯文本。位置参数与非空 stdin 同时存在是歧义错误。

```json
{"schema":"javdb.pipeline/v1","kind":"movie","ref":"SSIS-589","id":"9DGB5X","data":{},"meta":{}}
```

稳定 kinds：`movie`、`actor`、`series`、`maker`、`director`、`code`、`list`、
`account`、`comment`、`magnet`、`download`、`config_key`、`tag`、`error`。
消费者严格检查 kind 并优先使用合法 `id`；不兼容输入生成原位 `error` 信封。批处理保持输入
顺序、单项失败继续执行，最终非零并在 stderr 输出汇总。

输出默认使用人类可读文本。`--ndjson` 与 `--json` 互斥；只有显式指定对应 flag
才输出 JSON 或 NDJSON。显式 `--json` 单项保持既有 shape，多项输出信封数组。生产者命令
（如 `browse`、`tags`、`lists`、`rankings`、`top250`、`watched`、`want`、`recent`）
不读 stdin 且同样默认文本；使用 `--ndjson` 才逐条输出信封。

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
javdb rankings movies [--type TYPE] [--period day|week|month] [--has-magnets] [--json]
javdb rankings actors [--period day|week|month] [--json]
javdb rankings playback [--filter-by TYPE] [--period day|week|month] [--has-magnets] [--json]
javdb top250 [--zone ZONE] [--year YYYY] [--from RANK] [--page N] [--limit N] \
  [--ignore-watched] [--has-magnets] [--json]

javdb watched [--has-magnets]
javdb want [--has-magnets]
javdb recent [--has-magnets]
javdb collections actors|series|codes|makers|directors
javdb mark NUMBER --watched|--want [--score N] [--content TEXT] [--id]
javdb unmark NUMBER [--id]
```

`rankings movies --type` 与 `rankings playback --filter-by` 可用
`censored`、`uncensored`、`western` 或 `fc2`。CLI 将这些名称交给 SDK，由 SDK
映射为 App API 的数字排行分区值。三个排行命令的周期均使用 `day`、`week` 或 `month`，
内部会完成周期归一化。

`rankings movies`、`rankings playback` 与 `top250` 使用 `--json` 时输出 `{"movies":[...]}`；
`rankings actors` 输出 `{"actors":[...]}`。这些只含结果的对象会在 `--has-magnets`
过滤后输出。`magnets` 无需登录即可使用，token 失效时自动回退匿名请求。`top250` 需要登录。
`--best` 只从服务端返回的磁力集中选择，不会下载。
`mark`/`unmark` 会改写远程的看过/想看状态；`mark` 必须且只能传入
`--watched` 或 `--want` 之一。替他人或其他账号操作前必须确认。

## 版本与更新

```bash
javdb --version
javdb update [--check] [--prerelease] [--json]
```

`javdb --version` 对正式版输出两行（`javdb version 0.7.0 (2026-08-12)` 与 Release
URL，展示版本不带 `v`），开发版输出单行且不显示 Release URL。旧的
`javdb version --json` shim 仍保留供旧版更新器调用，但已从 help 与 completion 隐藏。

`update` 只在用户显式调用时执行，不会后台自动更新。`update --check` 只查询 GitHub Releases，
输出 `source`、`current_version`、`latest_version`、`latest_prerelease` 与 `update_available`。
只有 `--check` 可组合 `--json`，以获得该机器可读结果；不加 `--check` 时，仅在存在更高的所选版本后安装。

命令会保留安装渠道：Homebrew 使用 Formula，`go install` 使用精确 Release tag 重新安装，Release
压缩包只下载匹配平台的资产。压缩包安装先校验该 Release `release-manifest.json` 的 Ed25519
签名并绑定仓库/tag/平台，再按清单校验归档与解包二进制的 SHA-256，全部通过后才替换；下载的候选
二进制绝不执行。`--prerelease` 会纳入预发布 tag；Homebrew 安装无法安装该类 tag。`update` 会为
GitHub 请求独立解析 `--proxy` 与代理配置，并忽略 `--host`、`JAVDB_HOST` 和已配置的 JavDB
host，因为它不会访问 App API。

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
4. `download` 会写入本地文件：先取得明确输出路径，且不要替换已有文件。
5. 登录、刷新标签、改配置、选账号、`mark`/`unmark` 都是状态变更，执行前确认。

coding agent 的确认、凭据与错误处理规则见
[javdb-cli operator skill](../../skills/javdb-cli/SKILL.md)。
