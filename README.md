# javdb (Go)

Command-line client for [JavDB](https://javdb.com) **app JSON API**, written in Go.

Binary: `javdb` · Module: `github.com/FlanChanXwO/javdb-cli`

> Sibling of the Python prototype at `javdb-cli` (local tree). This Go tree is the target rewrite: no MCP, multi-account password login, full command parity in progress.

## Status

**P0 done:** signature, signed HTTP client (tls-client), multi-account auth, config, `auth` / `config` / `version`.

Remaining: search/detail/magnets → browse/entities → user lists → rankings/top250/lists (see plan).

## Build

```bash
go build -o build/javdb ./cmd/javdb
# or
sh scripts/build.sh
```

Requires Go 1.22+ (module uses current toolchain).

## Auth

```bash
javdb auth login -u USER -p PASS     # or interactive prompts
javdb auth list
javdb auth use <user_id>
javdb auth remove <user_id>
javdb auth check [--json]
```

Credentials: `~/.javdb-cli/auth.json` (0600) — username, password, JWT, multi-account by numeric `user_id`.

Config: `~/.javdb-cli/config.toml`

```bash
javdb config path
javdb config get
javdb config set auto_relogin true
javdb config set host mirror   # or main
```

Root flags: `--proxy URL`, `--host mirror|main`.

Login never prints the token. User id is taken from JWT claims (`id`) / `GET /api/v1/users`.

## Public SDK

```go
import "github.com/FlanChanXwO/javdb-cli/javdb"

c, err := javdb.New(javdb.WithHost(javdb.HostMirror))
token, err := c.Login(ctx, user, pass)
uid, name, err := c.ResolveUserID(ctx)
```

## Tests

```bash
go test ./...
```
