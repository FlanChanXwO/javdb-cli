# javdb-cli Copilot Instructions

本仓库的主规则在 [`AGENTS.md`](../AGENTS.md)。本文件只给 Copilot 提供短提示，避免补全时发明不存在的包、命令或 API。

## Project Shape

- Go module: `github.com/FlanChanXwO/javdb-cli`
- Binary: `cmd/javdb`
- CLI controller: `internal/cli`
- Public SDK facade: `sdk/` (`package javdb`)
- App protocol adapter: `internal/javdb/appapi`
- Protocol helpers: `internal/javdb/protocol/*`
- Config management: `internal/config`
- Local state: `internal/storage/auth`, `internal/storage/tags`
- Explicit update checks: `internal/update`

## Commands

```bash
go test ./...
sh scripts/build.sh
```

Do not suggest package-manager, frontend, database, Docker, or release commands unless the repository already contains that workflow.

## Guardrails

- Keep `cmd/javdb` thin; `internal/cli` owns Cobra, input, and output.
- Remote JavDB operations are exposed only through the top-level `sdk/` public SDK (`package javdb`); do not import `internal/javdb/appapi` or `internal/javdb/protocol/*` adapters directly from CLI code.
- Command, flag, JSON, config, or auth behavior changes must update the two locales of the CLI reference, README, and `skills/javdb-cli/`.
- Do not print passwords or JWTs or hide authentication, network, JavDB API, or filesystem errors behind empty success results.
- Do not add arbitrary truncation, retry limits, item caps, timeout behavior, or silent fallback without evidence and documentation.
