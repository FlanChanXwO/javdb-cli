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
The local output is `build/javdb` (`build/javdb.exe` on Windows).

## Test

```bash
go test ./...
go test -race ./...
go vet ./...
sh scripts/test-package-release.sh
sh scripts/test-homebrew-formula.sh
sh scripts/test-workflows.sh
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
scripts/build-release.sh
scripts/package-release.sh
skills/javdb-cli/       # agent operator skill + focused references
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

## Release and platform verification

The release contract covers exactly six native targets: `darwin/amd64`,
`darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`, and
`windows/arm64`. Release binaries are built with `CGO_ENABLED=0`,
`-trimpath`, and `-buildvcs=false`; each archive contains only the target
binary, `LICENSE`, and `README.md`.

To rehearse one target locally (without publishing anything):

```bash
mkdir -p dist
sh scripts/build-release.sh \
  --version 0.1.1 \
  --target darwin/arm64 \
  --output dist/javdb
sh scripts/package-release.sh \
  --binary dist/javdb \
  --version 0.1.1 \
  --target darwin/arm64 \
  --output-dir dist
tar -xzf dist/javdb-cli_0.1.1_darwin_arm64.tar.gz -C /tmp/javdb-smoke
/tmp/javdb-smoke/javdb version --json
```

`package-release.sh` refuses unsupported targets, unexpected binary names,
symbolic-link output paths, and existing asset names. It uses `7z` on the
Windows Git Bash runner because that image does not provide `zip`.

GitHub Actions mirrors the pixiv-cli release shape while remaining appropriate
for this pure-Go project:

1. **Quality gate** runs formatting, unit/race tests, vet, local build, and the package/workflow tests on every PR and `main` push.
2. **Platform packaged binary smoke** runs the test suite, builds, packages,
   extracts, and executes `javdb version --json` on all six native runners.
3. A tag `vX.Y.Z` is validated as immutable and reachable from `main`. Six
   native jobs test the tagged source; fresh jobs then rebuild the only assets
   allowed into the release.
4. The publisher creates a draft Release, compares its uploaded assets with
   the local verified set, publishes it, and renders the Homebrew formula from
   the same `checksums.txt`. The staging formula is installed and checked on
   macOS and Linux runners before any tap deployment.
5. Tap deployment is intentionally opt-in: set repository variable
   `HOMEBREW_TAP_DEPLOY_ENABLED=true` and put `HOMEBREW_TAP_DEPLOY_KEY` in the
   protected `release` environment. Without both, the Release and Formula
   verification complete but the deployment job is skipped.
