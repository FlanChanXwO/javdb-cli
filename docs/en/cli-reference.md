# javdb CLI reference

[Documentation](../index.md) · [简体中文](../zh-CN/cli-reference.md)

This is the public command contract for the `javdb` binary. Run
`javdb <command> --help` before automating a command; the help text is the
source of truth for the exact flags accepted by the installed version.

## Global options and configuration

Every remote command accepts these persistent options:

| Option | Meaning |
| --- | --- |
| `--proxy URL` | HTTP(S) proxy for this invocation. |
| `--host mirror\|main` | Select the App API host for this invocation. |

Configuration precedence is command-line options, then environment, then
`~/.javdb-cli/config.toml`, then built-in defaults. The configuration commands
are intentionally local state changes:

```bash
javdb config path
javdb config get [KEY]
javdb config set KEY VALUE
javdb config unset KEY
```

Supported keys are `host`, `https_proxy` (or `proxy`), `auto_relogin`, and
`lang`. `auto_relogin` is disabled by default. When explicitly enabled, an
expired JWT can trigger one re-login using the password already stored for the
default account.

## Authentication and local state

```bash
javdb auth login [-u USER] [-p PASS] [--use]
javdb auth list
javdb auth use USER_ID
javdb auth remove USER_ID
javdb auth check [--json]
```

- Omit `-u` or `-p` for interactive input; a TTY password prompt does not echo
  the password.
- `auth login` and `auth check` never print the JWT.
- Account data lives in `~/.javdb-cli/auth.json`; supported POSIX platforms use
  mode `0600`. Windows does not expose POSIX mode bits in the same way.
- `auth use` changes the default account, and `auth remove` deletes an account.
  Treat both as deliberate state changes.

Do not put a password or JWT in a command transcript, issue, chat, or source
file. Magnet, TOP250, and personal-list commands require the default account.

## Read-only discovery

```bash
javdb search KEYWORD [--zone ZONE] [--sort SORT] [--filter-by FILTER] \
  [--type TYPE] [--page N] [--limit N] [--has-magnets] [--json]
javdb detail NUMBER [--id] [--magnets] [--json]
javdb tags [--zone ZONE] [--refresh]
javdb browse [--zone ZONE] [--tag REF]... [--main FLAG]... [--year YYYY] \
  [--month MONTH] [--sort SORT] [--order asc|desc] [--page N] [--limit N] [--json]
```

`search` accepts `censored`, `uncensored`, `western`, `fc2`, or `all` for
`--zone`; `--type` can select `movie`, `code`, `series`, `actor`, `maker`,
`director`, or `list`. `detail --json` includes graph IDs that can be passed to
entity commands. `tags --refresh` downloads and rewrites the local public tag
cache, so it is not read-only local behavior.

`browse --tag` accepts a tag ID, English name, or Chinese name. Repeat
`--main` for server-side category masks. Use `--json` for programs; human output
uses tab-separated rows and is not a stable machine schema.

## Entity and list navigation

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

Entity options include zone, repeated tag/main filters, sorting, page/limit,
`--has-magnets`, and JSON output. `lists` without a subcommand reads the
authenticated user's lists; `list REF` is the entity-filmography command for a
public or user list.

## Magnets, rankings, and personal state

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

`magnets` and `top250` need authentication. `--best` chooses from the returned
magnet set; it does not download anything. `mark` and `unmark` change remote
watch/want state. `mark` requires exactly one of `--watched` or `--want`; obtain
confirmation before running either command for another person or account.

## Update and version

```bash
javdb update [--check] [--prerelease] [--json]
javdb version [--json]
```

`update` is explicit: it never runs in the background. `update --check` only
queries GitHub Releases and reports `source`, `current_version`,
`latest_version`, `latest_prerelease`, and `update_available`. Add `--json` only
with `--check` for that machine-readable result. Without `--check`, it installs
only when a newer selected release exists.

The command preserves the installation channel: Homebrew uses its Formula,
`go install` re-runs the exact release tag, and a Release archive downloads only
the matching platform asset. Archive installation verifies the asset SHA-256
from that Release's `checksums.txt` and runs the downloaded binary's
`version --json` before replacement. `--prerelease` includes prerelease tags;
Homebrew installations cannot install those tags. `--proxy` applies to GitHub
requests; `--host` does not, because update never contacts the JavDB App API.

Development builds (`version=dev`) deliberately refuse self-update. Install a
published release first. On Windows, a successful replacement leaves the prior
binary as a temporary `.old` file, which javdb removes on its next startup.

```bash
javdb version [--json]
```

`version --json` emits `version`, `commit`, and `build_date`. Commands that
support `--json` reserve stdout for a JSON result. A failed request returns a
non-zero exit status and a diagnostic on stderr; an upstream failure is not
represented as a fabricated empty result.

## Safe automation flow

1. Use `search --json` or `detail --json` to obtain a movie or graph ID.
2. Pass only the returned ID or explicit human-selected text to the next
   command; verify flags with `--help`.
3. Use `magnets --best --json` only after confirming that a magnet URI is in
   scope for the user.
4. Treat login, tag refresh, configuration edits, account selection, and
   mark/unmark operations as state changes and ask before performing them.

For coding-agent confirmation, credential, and error-handling rules, use the
[javdb-cli operator skill](../../skills/javdb-cli/SKILL.md).
