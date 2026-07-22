# Contributing

[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

Thanks for helping improve **javdb**.

## Development setup

```bash
git clone https://github.com/FlanChanXwO/javdb-cli.git
cd javdb-cli
go test ./...
sh scripts/build.sh
./build/javdb version --json
```

- Go version: see `go.mod` (currently 1.26.x).
- Do **not** commit `~/.javdb-cli/auth.json`, tokens, or passwords.
- Prefer offline unit tests; optional live smoke uses your local credentials and must never log secrets.

## Layout

```
cmd/javdb/          # binary entry
javdb/              # public SDK
internal/
  appapi/           # signed app API client
  cli/              # Cobra commands & printers
  config/           # config.toml paths & merge
  httpx/            # TLS client wrapper
  signature/        # request signing
  storage/auth/     # multi-account store
  storage/tags/     # tag taxonomy cache
  buildinfo/        # version ldflags
scripts/            # build helpers
skills/javdb-cli/    # agent operator skill
docs/               # user & dev docs (EN + ZH)
```

## Coding guidelines

1. Match existing CLI behavior and flag names where a Python prototype exists.
2. Keep printers and JSON shapes stable for agent/scripting use.
3. New endpoints: add unit tests for param builders first (TDD).
4. No reverse-engineering write-ups in `docs/` or README (product docs only).
5. Auth errors: surface clear messages; honor `auto_relogin` only when configured.

## Pull requests

1. Run `go test ./...`, `go test -race ./...`, `go vet ./...`, `sh scripts/build.sh`, `sh scripts/test-package-release.sh`, `sh scripts/test-homebrew-formula.sh`, and `sh scripts/test-workflows.sh`.
2. Keep commits focused; reference the feature in the message.
3. Update `CHANGELOG.md` / `CHANGELOG.zh-CN.md` for user-visible changes.
4. CI on `main` / PRs must stay green once workflows are enabled.

## License

By contributing, you agree that your contributions are licensed under the [MIT License](./LICENSE).
