---
name: javdb-cli-test
description: Select and run javdb-cli TDD, Go, SDK, CLI, pipeline, media, documentation, and release-tool checks. Use to reproduce failures, validate source or documentation changes, and report readiness while keeping routine verification offline and credential-free.
---

# Test javdb-cli

## Establish Red and scope

Inspect the diff, actual owner, and existing tests. Use the Go toolchain declared in `go.mod`; run from the repository root. Missing tools/dependencies are explicit environment blockers, not permission to install or weaken a gate.

For a feature or fix, run a focused behavioral test before implementation and confirm the expected failure. Build/import/fixture setup failures do not count as Red. Make the smallest implementation pass, then refactor with regression checks. Pure restructuring needs passing characterization before and after; any behavior correction needs its own failing test. Obtain an explicit exception when a required Red cannot be defined or run.

Use the existing Go test framework, `httptest`, temporary directories, and synthetic auth/HTTP fixtures. Test real CLI/SDK behavior rather than restating internal logic. Keep tests near their owner, name them by behavior, and table-drive equivalent cases when it improves clarity. Use an external test package where the public surface suffices; use same-package tests for a justified private seam rather than exporting internals solely for tests. Follow established same-stem/platform file conventions; do not create task-numbered regression files.

## Choose verification

Start with `go test ./path/to/owner -run '^TestName$' -count=1`, substituting the inspected package/test. Expand as needed:

| Change | Relevant checks |
| --- | --- |
| Documentation or agent/product instructions | Check English content, links, frontmatter, command examples, and `git diff --check`; run affected existing script/tool contracts |
| Local Go implementation | Focused and related package tests plus scoped `go vet`; `sh scripts/build.sh` when executable/build behavior is affected |
| Shared contract, public SDK, core behavior, or release candidate | `go test ./... -count=1`, `go vet ./...`, and `sh scripts/build.sh`, in addition to focused regression |
| Concurrent batch, auth/state, files, or core contracts | Add `go test -race ./... -count=1`; cover cancellation, partial failures, cleanup, and no-overwrite behavior |
| Metadata and PR verification | `go test ./tools/prmeta ./tools/verification -count=1` |
| Build/platform/release | `go test ./tools/release ./tools/platformmatrix -count=1`, `go test ./scripts/... -count=1`, and affected `scripts/test-*.sh` contracts |
| Image/HLS/TS/MP4 | [javdb-cli-media](../javdb-cli-media/SKILL.md), using complete and malformed synthetic fixtures |

Format affected Go with `gofmt` and inspect its diff. Follow `.pre-commit-config.yaml`; run installed hooks when applicable and report unavailable tooling rather than installing it silently. JavDB is a pure-Go build; no Rust or Pixiv staticlib step belongs here. The local binary is `build/javdb` (`.exe` on Windows).

The workflow's `scripts/changescope` policy owns docs-only classification; root `AGENTS.md` is not automatically exempt. Run the required checks for the actual classifier result. A fixture for workflow permissions is not proof of GitHub environment/secret settings or native platform execution.

## Contracts worth testing when touched

- Output: TTY text, default piped records, explicit legacy JSON, explicit `javdb.pipeline/v1` NDJSON, stable entity/list IDs, ordered per-item errors, and aggregate nonzero status. Keep the separate asset `TYPE<TAB>URL` stream intact.
- Identity/auth: exact number resolution before writes, known-ID bypass, optional-auth anonymous behavior, configured auto-relogin, and redacted errors. Use fake accounts only.
- State: CLI/environment/file/default precedence, fixed versus automatic host routing, private file publication, malformed cache behavior, and preservation of existing data.
- Updates: signature/origin/tag/platform binding, archive and extracted-binary hashes, no candidate execution, and preservation of the previous executable on failure.

## Offline versus live

Normal tests use fixtures and must not inspect `~/.javdb-cli/auth.json` or real account state. A real API request may refresh credentials, update route/tag caches, upload an image, or mutate remote marks. Run it only with explicit authorization, exact scope, and protected credentials. Do not reuse a production account because local tests lack a fixture, and do not mistake an offline pass for upstream acceptance.

For a failed gate, isolate the causal failure and run relevant safely independent phases that fail-fast hid. Rerun only after a relevant change or for a stated flaky/infrastructure hypothesis. Separate task regressions from baseline/environment failures; never edit a test to hide an unexplained mismatch.

Finish with exact commands and outcomes, tested commit, unrun/live/platform checks, and risks. Required blocked evidence remains blocked.
