# Contributing

[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

Thanks for improving `javdb-cli`. Read the [architecture guide](docs/maintainers/architecture.md)
and [development guide](docs/maintainers/development.md) before a broad change.

## Local setup

```bash
git clone https://github.com/FlanChanXwO/javdb-cli.git
cd javdb-cli
go test ./...
sh scripts/build.sh
./build/javdb version --json
```

- Use the Go version declared in `go.mod`.
- Never commit `~/.javdb-cli/auth.json`, a password, a JWT, a tag cache, or a
  machine-specific configuration file.
- Prefer offline tests. A live API check may use local credentials and must be
  explicitly authorized and free of secret output.

## Project map

```text
cmd/javdb/                         # binary entry
javdb/                             # public SDK facade
internal/cli/                      # Cobra input/output adapter
internal/javdb/appapi/             # App JSON API adapter
internal/javdb/protocol/{httpx,signature}/
internal/config/                   # configuration paths and merge
internal/storage/{auth,tags}/      # local state
internal/buildinfo/                # linker metadata
scripts/                           # build, package, and policy checks
skills/javdb-cli/                  # product operator skill
docs/en/, docs/zh-CN/              # localized public contracts
docs/maintainers/                  # architecture, development, ADR, agent rules
```

The CLI must use the public `javdb` facade for remote operations. Do not expose
protocol implementation paths as SDK API or create empty layers simply to copy
features that only exist in pixiv-cli.

## Change expectations

1. Preserve command names, flags, JSON fields, and text output unless a
   compatibility change is intentional and documented.
2. Write focused tests for behavior changes; keep pure filtering and parameter
   construction table-driven where practical.
3. Authentication failures must remain clear and must not reveal credentials.
4. Update both public locales, README, the operator skill, and routed
   maintainer docs when behavior changes.
5. Record user-visible additions, fixes, removals, or security changes in both
   changelogs under `Unreleased`.

## Before a pull request

```bash
go test ./...
go test -race ./...
go vet ./...
sh scripts/build.sh
sh scripts/test-package-release.sh
sh scripts/test-homebrew-formula.sh
sh scripts/test-workflows.sh
sh scripts/test-documentation.sh
sh scripts/test-architecture.sh
pre-commit run --all-files
```

Run the checks relevant to your change at minimum; run the full list before a
release-sensitive or broad refactor. Keep commits focused and discuss large or
compatibility-sensitive changes before implementation.

## License

By contributing, you agree that your contributions are licensed under the
[MIT License](LICENSE).
