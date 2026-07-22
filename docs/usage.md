# Usage

[English](usage.md) | [简体中文](usage.zh-CN.md)

Command-line reference for the `javdb` binary.

Global flags (all commands):

| Flag | Description |
|------|-------------|
| `--proxy URL` | HTTP(S) proxy (else `HTTPS_PROXY` / `ALL_PROXY` / config) |
| `--host mirror\|main` | API host; default **mirror** |

---

## Authentication

```bash
javdb auth login [-u USER] [-p PASS] [--use]
javdb auth list
javdb auth use <user_id>
javdb auth remove <user_id>
javdb auth check [--json]
```

- Missing `-u` / `-p` → interactive prompts (password is silent on TTY).
- Login **never prints** the JWT; only a short confirmation with username and user id.
- Multi-account store: `~/.javdb-cli/auth.json` (mode `0600` on POSIX platforms).
- Switch default account with `auth use` only (no per-command `--uid`).

### Config

```bash
javdb config path
javdb config get [key]
javdb config set <key> <value>
javdb config unset <key>
```

Keys: `host`, `https_proxy` / `proxy`, `auto_relogin`, `lang`.

Precedence: **CLI flags > environment > config.toml > defaults**.

`auto_relogin` (default `false`): when `true` or `JAVDB_AUTO_RELOGIN=1`, expired JWT triggers one silent re-login using the saved password.

---

## Search & catalog

### search

```bash
javdb search KEYWORD [--zone censored|uncensored|western|fc2|all]
  [--sort relevance|release|score|update|hit]
  [--filter-by can_play|magnets|subtitle|single]
  [--type movie|code|series|actor|maker|director|list]
  [--page N] [--limit N] [--has-magnets] [--json]
```

Default zone is `censored`. Use `--zone all` for cross-zone search.

### tags

```bash
javdb tags [--zone censored|uncensored|western|fc2] [--refresh]
```

Lists id / English / 中文. Cache: `~/.javdb-cli/tags-{zone}.json`. First run or `--refresh` downloads from the API.

### browse

```bash
javdb browse [--zone …] [--tag REF]... [--main p|m|c|s|i|v]...
  [--year YYYY] [--month M] [--sort hit|release|…] [--order asc|desc]
  [--page N] [--limit N] [--has-magnets] [--json]
```

`--tag` accepts id, English name, or 中文 (see `tags`).  
`--main m` ≈ downloadable / has magnets on the server mask.

### detail

```bash
javdb detail NUMBER [-i|--id] [--magnets] [--json]
```

Prints graph fields (series / maker / director / actors / tags) for agent navigation.  
`--magnets` requires login.

### magnets

```bash
javdb magnets NUMBER [-i] [--cnsub] [--hd] [--min-size SIZE]
  [--best] [--json]
```

Needs login. `--best` picks cnsub > hd > size. `--min-size` accepts `2000`, `4GB`, `500MB`.

---

## Entity filmography

```bash
javdb actor|series|maker|director|code|list REF
  [--zone] [--tag]... [--main]... [--sort release] [--order desc]
  [--page] [--limit] [--all] [--has-magnets] [--json]
```

`REF` is an id or name. `list` is a public/user **合集** (playlist) filmography (`filter_by` letter `l`).

---

## User lists (auth)

```bash
javdb watched [--has-magnets]
javdb want [--has-magnets]
javdb recent [--has-magnets]
javdb collections actors|series|codes|makers|directors
javdb mark NUMBER --watched|--want [--score N] [--content TEXT] [-i]
javdb unmark NUMBER [-i]
```

Exactly one of `--watched` / `--want` is required for `mark`.

---

## Rankings & TOP250

```bash
javdb rankings movies [--type censored|uncensored|western] [--period day|week|month] [--has-magnets]
javdb rankings actors [--period day|week|month]
javdb rankings playback [--filter-by censored|…] [--period …] [--has-magnets]

javdb top250 [--zone …] [--year YYYY] [--from RANK] [--page] [--limit]
  [--ignore-watched] [--has-magnets]
```

`top250` needs login. Omit `--zone`/`--year` for all-site TOP250.  
Actor periods are mapped internally (`day` → `daily`, etc.).

---

## 合集 (playlists)

```bash
javdb lists [--page] [--limit] [--sort-by created|name|movies_count|views_count|updated|default] [--json]
javdb lists show REF [--json]
javdb lists search KEYWORD [--zone all] [--page] [--limit] [--json]
javdb lists related NUMBER [-i] [--page] [--limit] [--json]
```

- `lists` (default) = **my** lists (auth; `sort-by` required by the API).
- Movies inside a list: `javdb list <id>` (entity command).

---

## Version

```bash
javdb version
javdb version --json   # {"version":"v0.1.0","commit":"…","build_date":"…"}
```

---

## Agent-oriented tips

1. `detail NUMBER --json` → follow `series_id` / actor ids → `series` / `actor` with `--main m --has-magnets`.
2. `magnets NUMBER --best --json` → take `magnet_uri`.
3. Prefer `--json` for scripting; text columns are tab-separated for humans.
4. Keep a logged-in default account for magnets / top250 / user lists.
