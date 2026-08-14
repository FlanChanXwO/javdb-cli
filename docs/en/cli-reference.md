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
| `--host auto\|mirror\|main\|URL` | Select the App API host for this invocation; `auto` (the default) immediately reuses a valid cached route and only re-selects the fastest startup candidate when validation fails. |

Configuration precedence is command-line options, then environment, then
`~/.javdb-cli/config.toml`, then built-in defaults. The configuration commands
are intentionally local state changes:

```bash
javdb config path
javdb config get [KEY]
javdb config set KEY VALUE
javdb config unset KEY
```

Supported keys are `host`, `https_proxy` (or `proxy`), `auto_relogin`,
`lang`, and the `reverse_search` scalars (`reverse_search.default_source`,
`reverse_search.cache`, `reverse_search.cache_ttl`,
`reverse_search.retries`, `reverse_search.retry_wait`,
`reverse_search.request_timeout`). `auto_relogin` is disabled by default. When
explicitly enabled, an expired JWT can trigger one re-login using the password
already stored for the default account.

With no key, `config get` prints the common keys on a TTY and reads a batch of
keys from stdin otherwise. `config get`/`config unset` also accept piped
`config_key` envelopes or plain key lines. `config set` always takes two
explicit arguments and never reads stdin. Reverse-search sources are
hand-edited TOML under `[[reverse_search.sources]]`; see
[Reverse image search](#reverse-image-search).

The first real command on a fresh machine creates `~/.javdb-cli/config.toml`
with only the common keys shown above; help, bare/parent commands, `version`,
completion, argument-validation failures, and `config unset` on a missing file
never create or overwrite it. The default `host` is `auto`: before each real
command the CLI verifies the cached route with one signed `/startup` request
and only re-runs full discovery when that fails, persisting the new host to
`~/.javdb-cli/route.json` (mode `0600`, storing only the verified host URL).
A corrupt cache, a failed re-selection, or a failed write is an explicit error,
never a silent fallback. Fix the host to `mirror`, `main`, or an absolute URL
to bypass route discovery.

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
file. Magnet commands work anonymously (using the saved token when available);
TOP250 and personal-list commands require the default account.

## Read-only discovery

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

`search` accepts `censored`, `uncensored`, `western`, `fc2`, or `all` for
`--zone`; `--type` can select `movie`, `code`, `series`, `actor`, `maker`,
`director`, or `list`. `detail --json` includes graph IDs that can be passed to
entity commands. `tags --refresh` downloads and rewrites the local public tag
cache, so it is not read-only local behavior.

See [Pipeline protocol](#pipeline-protocol) for the `--ndjson` output contract
shared by the commands above and below.

`browse --tag` accepts a tag ID, English name, or Chinese name. Repeat
`--main` for server-side category masks. Use `--json` for programs; human output
uses tab-separated rows and is not a stable machine schema.

`comments` calls the movie reviews endpoint exactly once for the selected page;
it never prefetches or follows another page. Its default is page `1` with a
page size of `20`; pass any positive `--page` and `--limit` when a different
single page is needed. `--json` preserves the complete review objects returned
for that page.

## Local media downloads

```bash
javdb download NUMBER [--id] [--thumbnail PATH] [--preview-image PATH] [--preview-video PATH]
```

Set at least one output flag. `--thumbnail` writes the detail thumbnail;
`--preview-image` writes only `preview_images[0]` (the first preview image) and
does not enumerate or fall through to later previews. `--preview-video` writes
the complete HLS preview stream to the given path, including AES-128 decryption
when the playlist requires it. Use a `.ts` path for the current transport-stream
previews.

Output paths support the `{number}` and `{id}` placeholders; a piped batch of
movie refs must use them (all expanded targets are preflighted for uniqueness,
existing files, and missing parent directories before anything is written).
The command creates new files only: it refuses an existing output path and does
not create missing parent directories. It accepts completed single-media HLS
playlists; master playlists, byte-range media, fragmented-MP4 media, and
unfinished/live playlists fail explicitly instead of producing a partial file.

## Reverse image search

```bash
javdb search IMAGE|URL|--image [--source NAME] [--no-cache] [--json|--ndjson]
javdb cache reverse-search [--source NAME] [--clear]
```

`javdb search` accepts a local JPEG/PNG/WEBP file (up to 8 MiB), an HTTP(S)
image URL, or binary image bytes on stdin; `--image` forces image mode, and an
existing file or HTTP(S) URL argument is auto-detected. The image is uploaded
as-is to the configured source: the built-in AVScan provider
(`https://avscan.cc/search`) or a declared external source, with at most three
total requests and 30s/60s backoff for HTTP 429, per-request timeouts, and
transient transport errors. Every candidate is linked to JavDB with strict
case-insensitive exact number matching (no first-hit fallback) and full detail
is returned; partial candidate failures finish the output and exit non-zero.

Configuration lives in `config.toml`:

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

Header values only support static text plus `${ENV:NAME}` references; missing
variables are reported by name only, never with their value. Source names are
unique and limited to letters, digits, `-`, and `_`; `builtin` is reserved.
Responses are cached under `~/.javdb-cli/reverse-search-cache` (mode `0600`,
keyed by source + image SHA-256, 30-day TTL); the cache never stores the
original image, auth headers, or JavDB details. `javdb cache reverse-search
--clear [--source NAME]` removes only reverse-search cache entries.

Privacy: reverse search uploads your image to the configured provider (built-in
AVScan by default). Image URLs may point to private networks; embedded SDK users
must enforce their own network boundary.

## Pipeline protocol

Commands that take a single positional ref also accept a non-TTY stdin batch.
Input is classified in fixed order: image magic, `javdb.pipeline/v1` NDJSON
envelopes, then plain text lines. Providing both a positional argument and
non-empty stdin is an ambiguity error.

```json
{"schema":"javdb.pipeline/v1","kind":"movie","ref":"SSIS-589","id":"9DGB5X","data":{},"meta":{}}
```

Stable kinds: `movie`, `actor`, `series`, `maker`, `director`, `code`, `list`,
`account`, `comment`, `magnet`, `download`, `config_key`, `tag`, `error`.
Consumers check the kind strictly and prefer a valid `id`; incompatible input
becomes an in-place `error` envelope. Batch processing preserves input order,
continues after item failures, and exits non-zero with a summary on stderr.

Output defaults to human-readable text. `--ndjson` and `--json` are mutually
exclusive; JSON or NDJSON is emitted only when its flag is explicit.
`--json` keeps the legacy single-item shape and emits a JSON array of envelopes
for batch input. Producers (e.g. `browse`, `tags`, `lists`, `rankings`,
`top250`, `watched`, `want`, `recent`) never read stdin and also default to
text; use `--ndjson` to emit one envelope per record.

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
`--has-magnets`, and JSON/pipeline output (`--json` or `--ndjson`). `lists`
without a subcommand reads the authenticated user's lists; `list REF` is the
entity-filmography command for a public or user list.

## Magnets, rankings, and personal state

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

`rankings movies --type` and `rankings playback --filter-by` accept
`censored`, `uncensored`, `western`, or `fc2`. The CLI sends these names through
the SDK, which maps them to the App API's numeric ranking-zone values. All three
ranking commands accept `day`, `week`, or `month`; period normalization is
handled internally.

`rankings movies`, `rankings playback`, and `top250` emit `{"movies":[...]}` with `--json`;
`rankings actors` emits `{"actors":[...]}`. These result-only objects are emitted
after any `--has-magnets` filtering. `magnets` works without login and falls back to anonymous when a saved token is
rejected. `top250` needs authentication. `--best` chooses from the returned
magnet set; it does not download anything. `mark` and `unmark` change remote
watch/want state. `mark` requires exactly one of `--watched` or `--want`; obtain
confirmation before running either command for another person or account.

## Version and update

```bash
javdb --version
javdb update [--check] [--prerelease] [--json]
```

`update` is explicit: it never runs in the background. `update --check` only
queries GitHub Releases and reports `source`, `current_version`,
`latest_version`, `latest_prerelease`, and `update_available`. Add `--json` only
with `--check` for that machine-readable result. Without `--check`, it installs
only when a newer selected release exists.

`javdb --version` prints a release build as two lines
(`javdb version 0.7.0 (2026-08-12)` plus the Release URL, without the leading
`v`) and a development build as one line; development builds never show a
Release URL. The legacy `javdb version --json` shim still exists for older
updaters but is hidden from help and completion.

The command preserves the installation channel: Homebrew uses its Formula,
`go install` re-runs the exact release tag, and a Release archive downloads only
the matching platform asset. Archive installation verifies the Release's
`release-manifest.json` Ed25519 signature, binds repository/tag/platform, and
checks both the archive and the extracted binary SHA-256 against the manifest;
the downloaded binary is never executed. `--prerelease` includes prerelease
tags; Homebrew installations cannot install those tags. `update` resolves
`--proxy` and proxy configuration independently for GitHub requests. It ignores
`--host`, `JAVDB_HOST`, and the configured JavDB host because it never contacts
the App API.

Development builds (`version=dev`) deliberately refuse self-update. Install a
published release first. On Windows, a successful replacement leaves the prior
binary as a temporary `.old` file, which javdb removes on its next startup.

Commands that support `--json` reserve stdout for a JSON result. A failed
request returns a non-zero exit status and a diagnostic on stderr; an upstream
failure is not represented as a fabricated empty result.

## Safe automation flow

1. Use `search --json` or `detail --json` to obtain a movie or graph ID.
2. Pass only the returned ID or explicit human-selected text to the next
   command; verify flags with `--help`.
3. Use `magnets --best --json` only after confirming that a magnet URI is in
   scope for the user.
4. Treat `download` as a local file write: obtain an explicit output path and
   do not replace an existing file.
5. Treat login, tag refresh, configuration edits, account selection, and
   mark/unmark operations as state changes and ask before performing them.

For coding-agent confirmation, credential, and error-handling rules, use the
[javdb-cli operator skill](../../skills/javdb-cli/SKILL.md).
