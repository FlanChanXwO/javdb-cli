# Development

[English](development.md) | [简体中文](development.zh-CN.md)

How to build and test **javdb** locally.

## Requirements

- Go version from `go.mod` (1.26.x)
- Network only for optional live smoke (not required for `go test`)

## Build

```bash
sh scripts/build.sh
# optional version embed:
VERSION=0.1.0 sh scripts/build.sh
./build/javdb version --json
```

`scripts/build.sh` sets:

```text
-X …/internal/buildinfo.Version
-X …/internal/buildinfo.Commit
-X …/internal/buildinfo.BuildDate
```

Release CI should pass the same ldflags with `CGO_ENABLED=0`.

## Test

```bash
go test ./...
go test -race ./...
go vet ./...
```

Unit tests are offline (param builders, printers, auth store, etc.).  
Do not commit credentials. Live checks:

```bash
javdb auth login   # once
javdb search SSIS-589 --limit 1
javdb detail SSIS-589
javdb magnets SSIS-589 --best
```

## Layout

```text
cmd/javdb/main.go       # entry → cli.Run
javdb/                  # public SDK (importable)
internal/appapi/        # HTTP client + endpoints
internal/cli/           # Cobra commands, printers
internal/config/        # paths, config.toml merge
internal/httpx/         # TLS fingerprint HTTP client
internal/signature/     # request signature header
internal/storage/auth/  # multi-account auth.json
internal/storage/tags/  # tag taxonomy files
internal/buildinfo/     # version metadata
scripts/build.sh
docs/                   # product docs (EN + ZH)
```

## Hosts

| Name | Base URL |
|------|----------|
| `mirror` (default) | `https://jdforrepam.com` |
| `main` | `https://javdb.com` |

Prefer mirror for direct connectivity. Use `--proxy` when targeting the main host.

## Configuration files

| Path | Notes |
|------|--------|
| `~/.javdb-cli/auth.json` | accounts + default_user_id; mode `0600` |
| `~/.javdb-cli/config.toml` | host, proxy, auto_relogin, lang |
| `~/.javdb-cli/device_uuid` | stable device id for public params |
| `~/.javdb-cli/tags-*.json` | public tag catalogs (not secret) |

## Style notes

- Keep CLI flag names stable for scripts and agents.
- Prefer small pure helpers (masks, filters) with table-driven tests.
- Auth failures must not dump tokens or passwords.
- Product documentation only under `docs/` (no reverse-engineering narratives).

## Release checklist (high level)

1. Green `go test` / race / vet / build  
2. Tag `vX.Y.Z`  
3. CI builds multi-arch archives + checksums  
4. Update Homebrew formula on `FlanChanXwO/homebrew-tap`  

See the project plan / goal tasks for the full pipeline.
