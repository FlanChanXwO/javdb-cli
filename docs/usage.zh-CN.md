# 使用说明

[English](usage.md) | [简体中文](usage.zh-CN.md)

`javdb` 命令行参考。

全局参数（所有命令）：

| 参数 | 说明 |
|------|------|
| `--proxy URL` | HTTP(S) 代理（否则读 `HTTPS_PROXY` / `ALL_PROXY` / 配置文件） |
| `--host mirror\|main` | API 主机；默认 **mirror** |

---

## 登录与账号

```bash
javdb auth login [-u USER] [-p PASS] [--use]
javdb auth list
javdb auth use <user_id>
javdb auth remove <user_id>
javdb auth check [--json]
```

- 省略 `-u` / `-p` 则交互输入（TTY 下密码不回显）。
- 登录**不会打印** JWT，只输出用户名与 user id。
- 多账号：`~/.javdb-cli/auth.json`（权限 `0600`）。
- 切换默认账号只用 `auth use`（没有按命令的 `--uid`）。

### 配置

```bash
javdb config path
javdb config get [key]
javdb config set <key> <value>
javdb config unset <key>
```

键：`host`、`https_proxy` / `proxy`、`auto_relogin`、`lang`。

优先级：**命令行参数 > 环境变量 > config.toml > 默认值**。

`auto_relogin`（默认 `false`）：为 `true` 或设置 `JAVDB_AUTO_RELOGIN=1` 时，JWT 过期会用已存密码静默重登一次。

---

## 搜索与目录

### search

```bash
javdb search KEYWORD [--zone censored|uncensored|western|fc2|all]
  [--sort relevance|release|score|update|hit]
  [--filter-by can_play|magnets|subtitle|single]
  [--type movie|code|series|actor|maker|director|list]
  [--page N] [--limit N] [--has-magnets] [--json]
```

默认分区 `censored`。跨区搜索用 `--zone all`。

### tags

```bash
javdb tags [--zone censored|uncensored|western|fc2] [--refresh]
```

输出 id / 英文 / 中文。缓存：`~/.javdb-cli/tags-{zone}.json`。首次或 `--refresh` 会从 API 拉取。

### browse

```bash
javdb browse [--zone …] [--tag REF]... [--main p|m|c|s|i|v]...
  [--year YYYY] [--month M] [--sort hit|release|…] [--order asc|desc]
  [--page N] [--limit N] [--has-magnets] [--json]
```

`--tag` 可用 id、英文名或中文名（见 `tags`）。  
`--main m` 表示服务端掩码里的「可下载 / 有磁力」一类主属性。

### detail

```bash
javdb detail NUMBER [-i|--id] [--magnets] [--json]
```

打印系列 / 厂牌 / 导演 / 演员 / 标签等图节点，方便 agent 跳转。  
`--magnets` 需要登录。

### magnets

```bash
javdb magnets NUMBER [-i] [--cnsub] [--hd] [--min-size SIZE]
  [--best] [--json]
```

需要登录。`--best` 优先 中字 > HD > 体积。`--min-size` 支持 `2000`、`4GB`、`500MB`。

---

## 实体片单

```bash
javdb actor|series|maker|director|code|list REF
  [--zone] [--tag]... [--main]... [--sort release] [--order desc]
  [--page] [--limit] [--all] [--has-magnets] [--json]
```

`REF` 为 id 或名称。`list` 表示用户/公开**合集**片单。

---

## 用户列表（需登录）

```bash
javdb watched [--has-magnets]
javdb want [--has-magnets]
javdb recent [--has-magnets]
javdb collections actors|series|codes|makers|directors
javdb mark NUMBER --watched|--want [--score N] [--content TEXT] [-i]
javdb unmark NUMBER [-i]
```

`mark` 必须且只能指定 `--watched` 或 `--want` 之一。

---

## 排行与 TOP250

```bash
javdb rankings movies [--type censored|uncensored|western] [--period day|week|month] [--has-magnets]
javdb rankings actors [--period day|week|month]
javdb rankings playback [--filter-by censored|…] [--period …] [--has-magnets]

javdb top250 [--zone …] [--year YYYY] [--from RANK] [--page] [--limit]
  [--ignore-watched] [--has-magnets]
```

`top250` 需登录。不传 `--zone`/`--year` 为全站 TOP250。  
演员榜周期会映射为 API 词表（`day` → `daily` 等）。

---

## 合集

```bash
javdb lists [--page] [--limit] [--sort-by created|name|movies_count|views_count|updated|default] [--json]
javdb lists show REF [--json]
javdb lists search KEYWORD [--zone all] [--page] [--limit] [--json]
javdb lists related NUMBER [-i] [--page] [--limit] [--json]
```

- 默认 `lists` = **我的**合集（需登录；API 要求 `sort-by`）。
- 合集内影片：`javdb list <id>`（实体命令）。

---

## 版本

```bash
javdb version
javdb version --json   # {"version":"v0.1.0","commit":"…","build_date":"…"}
```

---

## 给 Agent 的建议

1. `detail NUMBER --json` → 用 `series_id` / 演员 id → `series` / `actor --main m --has-magnets`。
2. `magnets NUMBER --best --json` → 取 `magnet_uri`。
3. 脚本优先 `--json`；人读用制表符文本列。
4. 磁力 / TOP250 / 用户列表需要已登录的默认账号。
