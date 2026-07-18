# javdb

[English](README.md) | [简体中文](README.zh-CN.md)

Unofficial command-line client for the [JavDB](https://javdb.com) **app JSON API**, written in Go.

| | |
|---|---|
| **Binary** | `javdb` |
| **Module** | `github.com/FlanChanXwO/javdb-cli` |
| **License** | [MIT](./LICENSE) |
| **Homebrew** | `brew install FlanChanXwO/tap/javdb-cli` (after release) |

This is a **client** for the app API. It is not a website scraper and does not include MCP.

## Disclaimer

**javdb is an unofficial third-party tool and is not affiliated with, endorsed by, or related to JavDB or its operators.**

- You are solely responsible for complying with JavDB’s terms of service and with the laws of your jurisdiction.
- Credentials (username/password) and session tokens are stored **locally** under `~/.javdb-cli/` at your own risk. Prefer a dedicated account; never commit secrets.
- The software is provided **as is**, without warranty of any kind. The authors are not liable for account bans, data loss, legal issues, or any other damages arising from use of this tool.
- Intended for personal / educational use. Do not use it to harass others or to abuse the service.

## Install

### Build from source

```bash
git clone https://github.com/FlanChanXwO/javdb-cli.git
cd javdb-cli
sh scripts/build.sh
./build/javdb version
```

Requires Go **1.26+** (see `go.mod`).

### go install

```bash
go install github.com/FlanChanXwO/javdb-cli/cmd/javdb@latest
# or pin a tag after the first release:
# go install github.com/FlanChanXwO/javdb-cli/cmd/javdb@v0.1.0
```

### Homebrew

```bash
brew install FlanChanXwO/tap/javdb-cli
```

## Quick start

```bash
# log in (interactive if flags omitted); never prints the JWT
javdb auth login -u USER -p PASS
javdb auth list
javdb auth check --json

# search / detail / magnets
javdb search SSIS-589 --limit 5
javdb detail SSIS-589
javdb magnets SSIS-589 --best --json

# browse & entities
javdb tags --zone censored
javdb browse --tag 巨乳 --main m --limit 10
javdb actor 山手梨愛 --main m --has-magnets
javdb list RZ8Bm --limit 5

# user lists (auth)
javdb watched
javdb want
javdb recent
javdb mark SSIS-589 --want

# rankings / TOP250 / 合集
javdb rankings movies --period week
javdb top250 --limit 20
javdb lists
javdb lists search 巨乳
```

Root flags: `--proxy URL`, `--host mirror|main` (default **mirror**).

Config & credentials:

| Path | Purpose |
|------|---------|
| `~/.javdb-cli/auth.json` | Multi-account username/password/token (mode `0600`) |
| `~/.javdb-cli/config.toml` | host, proxy, `auto_relogin`, lang |
| `~/.javdb-cli/tags-*.json` | Per-zone tag taxonomy cache |

```bash
javdb config set auto_relogin true   # optional silent re-login on JWT expiry
javdb config set host mirror
```

## Public Go SDK

```go
package main

import (
	"context"
	"fmt"

	"github.com/FlanChanXwO/javdb-cli/javdb"
)

func main() {
	c, err := javdb.New(javdb.WithHost(javdb.HostMirror))
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	if _, err := c.Login(ctx, "user", "pass"); err != nil {
		panic(err)
	}
	res, err := c.Search(ctx, "SSIS-589", javdb.SearchOptions{Limit: 5})
	if err != nil {
		panic(err)
	}
	fmt.Println(len(res.Movies()))
}
```

See [docs/sdk.md](./docs/sdk.md) for more.

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/index.md](./docs/index.md) | Doc map |
| [docs/usage.md](./docs/usage.md) | Command reference |
| [docs/development.md](./docs/development.md) | Build, test, layout |
| [docs/sdk.md](./docs/sdk.md) | Public package |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | How to contribute |
| [CHANGELOG.md](./CHANGELOG.md) | Release notes |

Chinese versions: `*.zh-CN.md` next to each file.

## Tests

```bash
go test ./...
go test -race ./...
go vet ./...
sh scripts/build.sh
```

## License

[MIT](./LICENSE) © FlanChanXwO
