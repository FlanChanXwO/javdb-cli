# Contributing to javdb-cli

English | [简体中文](CONTRIBUTING.zh-CN.md)

Thanks for helping improve `javdb-cli`. Focused bug reports, documentation fixes, tests, and well-scoped features are welcome.

## Before you start

- Search existing issues and pull requests before opening a duplicate.
- Discuss large features, public API changes, new dependencies, or authentication changes before implementation.
- Never include passwords, JWTs, `~/.javdb-cli/auth.json` contents, tag caches, machine-specific configuration, or private API responses in an issue, fixture, commit, or CI log.
- Keep changes focused. Unrelated cleanup is easier to review as a separate pull request.

## Development environment

The supported source build uses Go `1.26.3` and a standard Go toolchain; there are no C or native dependencies.

Build and test from the repository root:

```bash
go test ./...
sh scripts/build.sh
./build/javdb --version
```

See the [development guide (Simplified Chinese)](docs/maintainers/development.md) for opt-in real API tests, release gates, and platform details.

## Architecture guardrails

- `cmd/javdb` delegates to `internal/cli`; keep the binary thin and put Cobra, input, and output handling in `internal/cli`.
- Remote JavDB operations are exposed only through the top-level public `sdk/` SDK (`package javdb`); protocol code lives in `internal/javdb/appapi` and `internal/javdb/protocol/*`.
- `internal/config` manages local configuration; `internal/storage/auth` and `internal/storage/tags` manage local state; `internal/update` owns explicit update checks, source identification, and verified replacement.
- Keep files focused on one responsibility or a few tightly related responsibilities.

Read [the architecture guide (Simplified Chinese)](docs/maintainers/architecture.md) and the repository [AGENTS.md](AGENTS.md) before changing these boundaries.

## Develop with tests

Use a red-green-refactor loop for code changes:

1. Add a focused test that fails for the intended behavioral reason.
2. Implement the smallest coherent change that makes it pass.
3. Refactor without changing the verified public behavior.
4. Run the focused tests, then the relevant regression suite.

Test public behavior through the public boundary whenever practical. Do not hide real authentication, network, JavDB API, filesystem, or encoding failures behind empty success results or silent fallback. Do not add arbitrary timeouts, truncation, pagination caps, retry limits, or hidden downgrade paths.

Real JavDB App API canaries are opt-in. Never run them with a user's local account unless that user has explicitly authorized it; never put a real token on a command line that may be stored in shell history.

## Documentation

Update documentation in the same pull request when changing a command, flag, SDK API, configuration key, environment variable, output contract, authentication flow, proxy behavior, or known limitation.

- Keep `README.md` and `README.zh-CN.md` behaviorally aligned.
- Keep both locale versions under `docs/en/` and `docs/zh-CN/` behaviorally aligned; never use untranslated placeholder content.
- Update `docs/sdk.md` / `docs/sdk.zh-CN.md`, `docs/maintainers/architecture.md`, or `docs/maintainers/development.md` according to their documented responsibility.
- Release notes are maintained directly in the release-prep PR under `changelog/vX.Y.Z/{en.md,zh-CN.md}`. Keep English and Simplified Chinese entries aligned, include a PR or direct-commit source in every entry, and update both changelog indexes. See the [development guide](docs/maintainers/development.md) for the exact process.
- Check `skills/javdb-cli/` when CLI commands, flags, or safety semantics change.

Keep stable rules in one authoritative document and link to them elsewhere instead of copying large sections.

## Pull request checklist

Before requesting review:

- [ ] The change is focused and its user-visible behavior is explained.
- [ ] New or changed code has focused tests that first demonstrated the failure.
- [ ] `go test ./... -count=1` passes.
- [ ] `go vet ./...` passes.
- [ ] `sh scripts/build.sh` passes.
- [ ] The release-sensitive checks pass (`scripts/test-package-release.sh`, `test-homebrew-formula.sh`, `test-workflows.sh`).
- [ ] `python -m pre_commit run --all-files` passes when pre-commit is available.
- [ ] `git diff --check` passes.
- [ ] English and Simplified Chinese documentation are synchronized where required.
- [ ] No credential, local state, or machine-specific artifact is included.

Conventional Commits are recommended for commit messages, for example `feat(cli): add account selection` or `docs: clarify anonymous fallback`. The project does not require a CLA, DCO sign-off, or signed commits unless a future policy explicitly says otherwise.

## License

By contributing, you agree that your contribution may be distributed under the repository's [MIT License](LICENSE).
