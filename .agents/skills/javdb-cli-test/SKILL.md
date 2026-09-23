---
name: javdb-cli-test
description: Select and run javdb-cli TDD, Go, SDK, CLI, pipeline, media, documentation, and release-tool checks. Use to reproduce failures, validate source or documentation changes, and report readiness while keeping routine verification offline and credential-free.
---

# Test javdb-cli

## Establish Red and scope

Inspect the diff, actual owner, and existing tests. Use the Go toolchain declared in `go.mod`; run from the repository root. Missing tools/dependencies are explicit environment blockers, not permission to install or weaken a gate.

For a feature or fix, select an existing test that exposes the missing behavior, or extend it only where coverage is missing, and confirm the expected failure before implementation. Build/import/fixture setup failures do not count as Red. Make the smallest implementation pass, then refactor with regression checks. Pure restructuring reuses passing characterization before and after; any behavior correction needs failing evidence. Obtain an explicit exception when an applicable Red requirement cannot be met.

Use the existing Go test framework, `httptest`, temporary directories, and synthetic auth/HTTP fixtures. Test real CLI/SDK behavior rather than restating internal logic. Keep tests near their owner, name them by behavior, and table-drive equivalent cases when it improves clarity. Use an external test package where the public surface suffices; use same-package tests for a justified private seam rather than exporting internals solely for tests. Follow established same-stem/platform file conventions; do not create task-numbered regression files.

## Decide whether test code must change

Before adding a test, name the observable contract or credible failure that existing tests do not protect. Inspect their assertions, not just their names or coverage percentage. Prefer running an existing test, then extending a relevant case/table, then adding a new test only for an uncovered scenario. Record the choice briefly in the existing handoff, not a new test-plan file.

- Test behavior, not the existence of each function or file. A getter, forwarding wrapper, simple field projection, or newly extracted helper needs no separate test when its behavior is already protected at the calling boundary. Short security, parsing, or persistence code can still carry independent risk and require coverage.
- For pure moves, renames, or extraction, run existing characterization before and after. Do not invent a behavior change, alter an assertion to manufacture Red, or duplicate a test solely because a helper appeared.
- Test at the narrowest stable boundary that catches the defect. Extra CLI/SDK/adapter layers earn tests for distinct serialization, validation, permissions, or failure semantics, not repeated assertions of the same mapping. Mock real external boundaries rather than recreating the implementation in a test.
- Keep fixtures and assertions direct. Use table-driven cases only when they share setup and semantics; avoid a generic fixture builder, mock hierarchy, assertion DSL, or new framework for a small case. Do not test standard-library behavior instead of the project's own contract.
- Ordinary prose comments, headings, and internal code layout do not need exact-text or AST/regex tests. Machine-consumed directives, generated contracts, parsers, and security boundaries may need targeted existing checks; distinguish those from style preferences.
- Remove or consolidate tests only within the authorized scope and after identifying the same contract protected by retained checks. Preserve regression inputs and run the retained checks; fewer lines alone do not justify deleting evidence.

Finish adding tests when changed contracts and credible failure paths are protected. There is no per-function, per-file, test-count, or invented coverage-percentage target. TDD governs the order of evidence, not the quantity of new test code; mandatory repository gates still apply.

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

CI scope is determined by `.github/ci-change-scope.gitignore` through `scripts/classify-change-scope.sh`. `ci.yml` owns Quality; trusted `pr-metadata.yml` classifies PRs and dispatches base-ref Platform/Container workers. Required smoke gates are Check Runs on the exact PR head and use a real skipped conclusion when their scope is not required. Ordinary branch/main pushes do not run CI; matching tags retain the current Quality/release path. Run the required checks for the actual classifier result. A fixture for workflow permissions is not proof of GitHub environment/secret settings or native platform execution.

## Contracts worth testing when touched

- Output: TTY text, default piped records, explicit legacy JSON, explicit `javdb.pipeline/v1` NDJSON, stable entity/list IDs, ordered per-item errors, and aggregate nonzero status. Keep the separate asset `TYPE<TAB>URL` stream intact.
- Identity/auth: exact number resolution before writes, known-ID bypass, optional-auth anonymous behavior, configured auto-relogin, and redacted errors. Use fake accounts only.
- State: CLI/environment/file/default precedence, fixed versus automatic host routing, private file publication, malformed cache behavior, and preservation of existing data.
- Updates: signature/origin/tag/platform binding, archive and extracted-binary hashes, no candidate execution, and preservation of the previous executable on failure.

## Offline versus live

Normal tests use fixtures and must not inspect `~/.javdb-cli/auth.json` or real account state. A real API request may refresh credentials, update route/tag caches, upload an image, or mutate remote marks. Run it only with explicit authorization, exact scope, and protected credentials. Do not reuse a production account because local tests lack a fixture, and do not mistake an offline pass for upstream acceptance.

For a failed gate, isolate the causal failure and run relevant safely independent phases that fail-fast hid. Rerun only after a relevant change or for a stated flaky/infrastructure hypothesis. Separate task regressions from baseline/environment failures; never edit a test to hide an unexplained mismatch.

Finish with exact commands and outcomes, tested commit, unrun/live/platform checks, and risks. Required blocked evidence remains blocked.
