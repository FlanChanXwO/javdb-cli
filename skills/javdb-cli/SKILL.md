---
slug: javdb-cli
version: 0.8.1
displayName: JavDB CLI
summary: Safely operate JavDB through the javdb-cli binary for discovery, authenticated lists, and explicit state changes.
license: MIT
homepage: https://github.com/FlanChanXwO/javdb-cli
tags: [javdb, cli, agent]
name: javdb-cli
description: Operate the installed javdb CLI to search the JavDB App API, inspect movie and entity references, read lists, and perform explicitly authorized account, configuration, mark, or preview-asset actions. Use only for an explicit JavDB or javdb-cli task, a javdb command, or a clear JavDB reference with an operation request. Do not trigger for generic search, adult-content discovery, or downloading. Verify syntax with javdb COMMAND --help.
---

# JavDB CLI Operator

Drive the installed `javdb` binary, not a guessed API or a source checkout. Read only the reference needed for the user's operation; no repository-maintenance skill or personal agent configuration is required.

## Preflight

1. Run `javdb --version`. If missing or not executable, report the blocker. Install only on an explicit installation/repair request using [install.md](references/install.md).
2. Inspect the relevant `javdb COMMAND --help` before execution. Do not list accounts on every turn; use `auth list` only for an account decision or explicit request. A stored account is not proof of a valid token.
3. Use networked `javdb auth check --json` only when identity validation is needed. Normal discovery does not authorize login, configuration repair, or host changes.

## Safety and authorization

- `~/.javdb-cli/auth.json` contains usernames, passwords, and JWTs. Never read, print, upload, quote, or summarize its contents. Supported POSIX systems use private `0600` permissions.
- For login and credential handling, read [auth.md](references/auth.md). Do not launch a hidden prompt in an agent terminal the user cannot type into. Never request an undisclosed password in chat as a workaround.
- Account/config changes, `mark`/`unmark`, cache clearing/refresh, updates, and file downloads require the user's explicit current target and action. Prior authorization does not extend to a new target. Reuse an already explicit instruction rather than asking the same question again.
- Use the App API. Do not substitute browser cookies or scraping after an error. Existing optional-auth anonymous retry for magnets and explicitly enabled `auto_relogin` are documented product behavior, not permission to add other fallback or retry steps.
- Image search uploads the image to AVScan or a configured provider. Explain that privacy effect and use only an authorized source. Media assets are previews/thumbnails, not full movies or magnet-target downloads.

## Choose the operation

| Task | Commands and scope |
| --- | --- |
| Read discovery/details | `search`, `detail`, `comments`, `magnets`, entity commands, `rankings`, `tags`, `browse`; see [discover.md](references/discover.md) |
| Read authenticated lists | `top250`, `watched`, `want`, `recent`, `collections`, default `lists`; require the selected local account |
| Diagnose identity | `auth list`, `auth check`; only when needed |
| Change remote/local state | `mark`, `unmark`, `auth login/use/remove`, `config set/unset`; see [state.md](references/state.md) |
| Save media assets | `assets list` then a scoped pipe into `assets download`; see [media.md](references/media.md) |
| Inspect/update installation | `update --check --json` is read-only; actual `update` requires explicit authorization |
| Manage caches | `tags --refresh` rebuilds tags; `cache reverse-search --clear` clears only reverse-search cache; use only for the requested cache task |

`magnets` and `detail --magnets` can run anonymously; a configured account is used when present, and a rejected optional token falls back to anonymous inside the CLI. Do not apply that behavior to authenticated list or write commands.

## Output and input contracts

Use the user's requested scope. Supply a positive `--limit` only when supported and appropriate to that scope; never add an arbitrary limit, page cap, timeout, or retry to save context. `--all` is command-specific and only for an explicit exhaustive traversal. `comments` reads one selected page (default page 1, limit 20), not an automatic all-pages loop.

TTY stdout defaults to human-readable text. Piped stdout normally uses stable text references/URIs; **explicit `--ndjson`** selects `javdb.pipeline/v1` envelopes. `--json` retains the command's aggregate/legacy shape where applicable and is mutually exclusive with `--ndjson`. Inspect exit status before parsing: errors can be plain stderr and empty stdout.

Most data commands accept non-TTY batch input. Explicit positional arguments plus non-empty stdin are ambiguous and fail. Login, password prompts, and `config set` are outside this generic pipeline path. Batch failures can preserve successful items and still exit nonzero; report both rather than calling them empty results.

Movie/list/entity envelopes carry their actual kind and stable ID; list/entity payloads live in `data.list`/`data.entity`. Missing stable IDs fail, while a missing display name can use the ID as `ref`. A known movie ID bypasses number resolution; never infer one from a title. Assets use a separate `TYPE<TAB>URL` stream.

```text
javdb search "ABC-123" --json
javdb search "QUERY" --type actor --json
javdb detail ABC-123 --json
javdb detail MOVIE_ID --id --json
javdb comments ABC-123 --page 1 --limit 20 --json
javdb lists search "QUERY" --zone all --ndjson
javdb list LIST_ID --json
javdb lists related ABC-123 --ndjson
javdb search ABC --ndjson | javdb detail
javdb rankings movies --type fc2 --period week
javdb update --check --json
```

Replace placeholders only with verified values. Examples illustrate syntax, not permission to perform extra requests.

## Image search and integrated magnet search

`javdb search IMAGE_OR_URL [--source NAME] [--no-cache]` accepts JPEG/PNG/WEBP up to the existing 8 MiB limit and uploads to the selected provider. Candidates are resolved to JavDB through strict movie-number matching; one failed candidate can leave other output and a nonzero exit. Do not hide the failed candidate.

Reverse-search caching defaults to 30 days, keyed by source and original-image SHA-256. `--no-cache` bypasses it for one search. `javdb cache reverse-search [--source NAME] [--clear]` addresses that cache only; do not delete unrelated local state.

`javdb search QUERY --magnets N` combines movie search, filtering, sorting, and magnet lookup. `--cnsub`, `--hd`, and non-negative `--min-size` filter before sorting; zero is valid and negative fractional sizes fail. `N=0` requests all and positive N selects the first N. Text output contains magnet URIs; NDJSON contains `kind=magnet`. This option does not download the targets and is only for movie search.

## Configuration and routing

Precedence is CLI flags, environment, `config.toml`, then defaults. `--proxy URL` and `--host auto|mirror|main|URL` affect one invocation. Supported proxy schemes are HTTP/HTTPS/SOCKS4/SOCKS4A/SOCKS5/SOCKS5H; a host is required and SOCKS needs an explicit port. An explicitly blank proxy is an error, not a direct-connection override.

CLI `host=auto` verifies and reuses `route.json`; when invalid, it discovers startup candidates and selects a route. Fixed `mirror`, `main`, or absolute URLs bypass discovery. Do not confuse the CLI default with the SDK constructor's mirror default or persist an unreviewed URL.

Actual update resolves its own Release transport, ignoring data host selectors. It detects Homebrew, `go install`, or Release-archive installation; development builds refuse self-update. Use `--prerelease` only on request, and not for Homebrew. See [install.md](references/install.md) for verification boundaries and [troubleshooting.md](references/troubleshooting.md) for failures.
