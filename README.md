<div align="center">

# javdb-cli

**JavDB CLI · Go SDK · Agent skill**

[English](README.md) · [简体中文](README.zh-CN.md)

<p><a href="https://github.com/FlanChanXwO/javdb-cli/actions/workflows/ci.yml"><img alt="Quality gate" src="https://github.com/FlanChanXwO/javdb-cli/actions/workflows/ci.yml/badge.svg"></a> <a href="https://github.com/FlanChanXwO/javdb-cli/actions/workflows/platform-smoke.yml"><img alt="Platform smoke" src="https://github.com/FlanChanXwO/javdb-cli/actions/workflows/platform-smoke.yml/badge.svg"></a> <a href="https://github.com/FlanChanXwO/javdb-cli/releases/latest"><img alt="Release" src="https://img.shields.io/github/v/release/FlanChanXwO/javdb-cli?style=flat-square"></a> <a href="go.mod"><img alt="Go" src="https://img.shields.io/github/go-mod/go-version/FlanChanXwO/javdb-cli?style=flat-square"></a> <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/FlanChanXwO/javdb-cli?style=flat-square"></a> <img alt="Views" src="https://hits.sh/github.com/FlanChanXwO/javdb-cli.svg?style=flat-square&amp;label=views"></p>

[Install](#install) · [Quick start](#60-second-quick-start) · [Interfaces](#choose-your-interface) · [Documentation](#documentation) · [Contributing](CONTRIBUTING.md)

</div>

`javdb-cli` is an independent, unofficial command-line client and Go SDK for the
[JavDB](https://javdb.com) App JSON API. It gives people, coding agents, and Go
applications one consistent capability surface for search, catalog navigation,
movie details, magnets, rankings, and authenticated user lists. It is not a
website scraper, does not include MCP, and is not affiliated with, endorsed by,
or related to JavDB or its operators. Use it only in accordance with JavDB's
terms and the laws that apply to you.

## Why javdb-cli?

- **One capability surface** — the CLI and public Go SDK cover search, detail,
  tags, browsing, entity filmographies, magnets, rankings, TOP250, collections,
  and authenticated watch/want data.
- **API client, not a scraper** — commands use the App JSON API with explicit
  host and proxy selection; failures remain visible rather than becoming
  fabricated empty results.
- **Useful agent navigation** — JSON output, stable graph IDs from `detail`,
  and an included [operator skill](skills/javdb-cli/SKILL.md) support safe,
  multi-step automation.
- **Local multi-account credentials** — username/password/token data stays in
  `~/.javdb-cli/auth.json` with mode `0600` on POSIX platforms; normal command
  output never prints the JWT.
- **Deliberate state changes** — watch/want marks, default-account changes,
  configuration writes, and tag-cache refreshes are explicit CLI operations.
- **Native release evidence** — macOS, Linux, and Windows on amd64 and arm64
  are built, packaged, extracted, and smoke-tested by CI.

## Install

### Homebrew (recommended on macOS and Linux)

```bash
brew install FlanChanXwO/tap/javdb-cli
```

Upgrade later with:

```bash
brew update
brew upgrade javdb-cli
```

The Formula is updated after a verified release when tap deployment is enabled.

### Go

Use an exact published tag:

```bash
go install github.com/FlanChanXwO/javdb-cli/cmd/javdb@v0.1.1
```

### Release archive or source build

Download the archive for your platform from
[GitHub Releases](https://github.com/FlanChanXwO/javdb-cli/releases), verify it
against the accompanying `checksums.txt`, and place `javdb` (`javdb.exe` on
Windows) in a directory on your `PATH`.

To build the checkout, install the Go version declared in `go.mod` and run:

```bash
sh scripts/build.sh
./build/javdb version --json
```

The release contract covers macOS, Linux, and Windows on amd64 and arm64. See
the [development guide](docs/maintainers/development.md#构建打包与平台)
for reproducible target builds and archive contents.

### Install with a coding agent

Copy this prompt into Codex, Claude Code, Cursor, or another local coding agent
with terminal access:

```text
Install the latest stable javdb-cli from https://github.com/FlanChanXwO/javdb-cli for this machine. Detect the operating system and architecture, download only official GitHub Release assets, require the matching published SHA-256 from checksums.txt before installing, ask before creating or changing any PATH directory, never read or output ~/.javdb-cli/auth.json or credentials, verify with javdb version --json, and report the installed version plus every changed file.

Also install the complete skills/javdb-cli/ directory from the same stable release tag into the agent skills directory that I confirm. Do not guess that skills path, do not use the main branch for the skill, and preserve all reference files.
```

## 60-second quick start

```bash
# Sign in interactively; the CLI never prints the JWT.
javdb auth login
javdb auth check --json

# Find a movie and inspect graph IDs for the next navigation step.
javdb search SSIS-589 --limit 5 --json
javdb detail SSIS-589 --json

# Browse a tag and request a filtered magnet list (authentication required).
javdb browse --tag 巨乳 --main m --limit 20 --json
javdb magnets SSIS-589 --cnsub --hd --json
```

Run `javdb --help` or read the [complete command reference](docs/en/cli-reference.md) for
all commands, flags, configuration keys, and authentication requirements.

## Choose your interface

### CLI

Use tabular output interactively and `--json` when a command exposes it and an
agent or script needs stable fields:

```bash
javdb rankings movies --period week
javdb actor 山手梨愛 --main m --has-magnets --json
javdb lists search 巨乳 --zone all --json
```

The global `--proxy URL` and `--host mirror|main` flags affect only that command.
Use `javdb config get` before relying on persisted settings.

### Go SDK

```go
c, err := javdb.New(javdb.WithHost(javdb.HostMirror))
if err != nil {
	panic(err)
}

res, err := c.Search(context.Background(), "SSIS-589", javdb.SearchOptions{Limit: 5})
if err != nil {
	panic(err)
}
fmt.Println(len(res.Movies()))
```

Import `github.com/FlanChanXwO/javdb-cli/javdb`. The [SDK guide](docs/en/sdk.md)
documents public models, client options, and caller responsibilities.

### Agent skill

The repository ships [skills/javdb-cli](skills/javdb-cli/SKILL.md), a dedicated
operator skill for coding agents. It defines credential handling, confirmation
boundaries, command-scoped flags, JSON/error handling, and search-to-detail
navigation. Load it only for explicit JavDB work, and verify each command's
flags with `javdb <command> --help`.

## Authentication and credential safety

`javdb auth login` is the recommended setup. It keeps the username, password,
and session token in the local multi-account store
`~/.javdb-cli/auth.json` (mode `0600` where POSIX permissions exist). Do not
commit, print, paste, or upload that file, passwords, or JWTs.

```bash
javdb auth list
javdb auth use USER_ID
javdb auth check --json
```

`javdb auth check` verifies the default token without printing it. Token expiry
is surfaced clearly by default. `auto_relogin` is opt-in and uses the saved
password for one re-login attempt; enable it only when you accept that behavior:

```bash
javdb config set auto_relogin true
```

Magnet, TOP250, and user-list commands need the default authenticated account.
`mark`/`unmark`, account changes, and `config set`/`unset` modify server or
local state and should be used deliberately.

## Documentation

| Guide | Use it for |
| --- | --- |
| [CLI reference](docs/en/cli-reference.md) | Commands, flags, authentication, configuration, and common workflows |
| [Agent operator skill](skills/javdb-cli/SKILL.md) | Safe agent routing, secrets, writes, discovery, and errors |
| [Go SDK](docs/en/sdk.md) | Public client, models, and options |
| [Architecture (Simplified Chinese)](docs/maintainers/architecture.md) | Package boundaries and runtime flow |
| [Development (Simplified Chinese)](docs/maintainers/development.md) | Toolchain, tests, platform builds, packaging, and releases |
| [Documentation map](docs/index.md) | Localized public contracts and maintainer guides |
| [Contributing](CONTRIBUTING.md) | Local quality gates and contribution rules |
| [Changelog](CHANGELOG.md) | User-visible changes |

## Contributing

Bug reports, documentation fixes, tests, and focused features are welcome.
Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request; discuss
large or compatibility-sensitive changes first.

## About the views badge

The `hits.sh` badge counts image requests, not unique people. Repeated visits,
bots, and caches can affect it; loading the badge also makes a normal third-party
image request to `hits.sh`. Both language pages use the same badge URL.

## License

[MIT](LICENSE) © FlanChanXwO
