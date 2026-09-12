# Movie resolution correctness plan

Branch: `fix/movie-resolution-correctness`
Base: `main@1b21ca94a3d11008ce6028c670ea36fc2fbac3c9`

## Goal

Make movie-number resolution fail closed instead of silently selecting the first search hit, and repair the state-changing movie consumers that currently mishandle pipeline IDs or pre-read stdin.

This branch owns **movie identity correctness**. It must not absorb unrelated pipeline protocol, list fan-out, config output, magnet filter, or download-command naming work.

## User-visible problems covered

1. `ResolveMovieID` currently prefers an exact case-insensitive number match but falls back to the first search result when no exact number is present. A mistyped number can therefore resolve to the wrong movie.
2. `mark` checks whether non-TTY stdin has content with a temporary `bufio.Reader.Peek(1)`. That reader may buffer bytes from the underlying pipe and then be discarded, so the real batch reader can observe EOF or truncated input.
3. `mark` and `unmark` use `pipeline.ConsumerRef(input)` but can still call `ResolveMovieID` when the envelope already carries an internal movie ID. In that path the internal ID is incorrectly treated as a printed number.

## Non-goals

- Do not rename or redesign `download` in this branch.
- Do not modify `lists related`; that belongs to `fix/cli-contract-correctness`.
- Do not change pipeline envelope kinds, output cardinality, JSON/NDJSON rendering, or shared empty-input diagnostics.
- Do not add BitTorrent, 115, aria2, qBittorrent, or any downloader backend.
- Do not change release notes or `changelog/vX.Y.Z/`; release-prep remains a separate process.

## Correctness contract

### Printed movie numbers

A caller that supplies a printed number such as `SSIS-589` must resolve only an exact, case-insensitive number match after trimming surrounding whitespace.

If there is no exact match, return a not-found style error. Never return the first fuzzy search hit.

If the API returns more than one distinct exact match, fail rather than choose silently. Reuse the existing strict semantics already represented by `ResolveMovieIDExact` / `ResolveNumberExact` instead of creating a second definition of exactness.

### Pipeline movie envelopes

For an input envelope with `kind=movie` and a non-empty `id`, consumers must treat `id` as authoritative and skip number resolution. `--id` applies only to raw positional or line input that does not already carry an envelope ID.

### `mark` stdin

`mark` must never probe stdin through a disposable buffered reader before handing the same stream to the batch runner.

The command always requires exactly one of `--watched` or `--want`. Validate that requirement without consuming stdin. Missing input remains the shared runner's responsibility.

## Implementation tasks

### Task 1 — lock down strict resolver behavior with failing tests

Add regression tests before implementation changes.

Primary files:
- `internal/javdb/appapi/endpoint/movie/resolve_test.go` or the existing resolver test file
- `sdk/reversesearch_test.go` only if needed to prove public facade consistency; do not duplicate endpoint tests unnecessarily

Required cases:
- exact case-insensitive match succeeds;
- surrounding whitespace is accepted;
- fuzzy-only results return an error instead of the first result ID;
- no results return an error;
- duplicate exact matches fail deterministically;
- existing uncensored/western/fc2 resolution continues to use zone `all`.

The first test run must demonstrate the fuzzy-fallback failure before production code is changed.

### Task 2 — unify `ResolveMovieID` with strict resolution

Primary files:
- `internal/javdb/appapi/endpoint/movie/resolve.go`
- `sdk/movie.go` only for comments/signature-adjacent documentation if needed; do not rename the public method

Preferred implementation:
- make `ResolveMovieID` use the same strict exact-match rule as `ResolveMovieIDExact`;
- avoid maintaining two subtly different match algorithms;
- preserve the existing public function names and signatures;
- preserve `zone=all` behavior;
- do not introduce a compatibility flag for first-hit fallback.

If code reuse requires a small internal helper, keep it local to the movie resolver package. Do not create a repository-wide abstraction.

### Task 3 — fix `mark` stdin consumption

Primary files:
- `internal/cli/commands/mark/mark.go`
- `internal/cli/commands/mark/mark_test.go`

Add a regression test using a non-TTY reader containing at least one movie line. The test must prove the stream reaches the batch runner intact.

Remove the disposable `bufio.Reader.Peek` pattern. Prefer unconditional validation that exactly one of `--watched` / `--want` is selected before runner execution.

Do not modify shared `pipeline.BatchRunner` in this task.

### Task 4 — fix authoritative envelope-ID handling in `mark` and `unmark`

Primary files:
- `internal/cli/commands/mark/mark.go`
- `internal/cli/commands/mark/mark_test.go`
- `internal/cli/commands/unmark/unmark.go`
- `internal/cli/commands/unmark/unmark_test.go`

Required behavior:
- envelope with non-empty `id`: call the remote mutation using that ID directly;
- envelope without `id`: resolve its ref as a printed number unless positional/raw `--id` explicitly says otherwise;
- positional `--id` behavior remains unchanged;
- error-envelope passthrough behavior remains owned by shared pipeline code and must not be reimplemented here.

Tests should detect accidental calls to search/resolve when an envelope ID is already present.

### Task 5 — documentation sync within this branch's owned anchors

Because resolver semantics are public SDK behavior and affect movie-number commands, update only the resolver-related wording in:
- `docs/en/sdk.md`
- `docs/zh-CN/sdk.md`
- `docs/en/cli-reference.md` and `docs/zh-CN/cli-reference.md`: only the common movie-number resolution / mutation behavior paragraph, not local-media-download or pipeline sections
- `README.md` / `README.zh-CN.md`: only a concise correctness note if there is an existing behavior/safety paragraph; do not edit the command inventory owned by the download branch
- `skills/javdb-cli/SKILL.md`: only the rule describing exact movie-number targeting for mutations; do not edit media-download command names

Do not reflow unrelated paragraphs. Shared documentation files are intentionally partitioned by anchor so the three parallel branches can merge cleanly.

## Files owned by this branch

Production ownership:
- `internal/javdb/appapi/endpoint/movie/resolve.go`
- resolver tests in the same package
- `internal/cli/commands/mark/**`
- `internal/cli/commands/unmark/**`
- `sdk/movie.go` only if needed for resolver docs/comments

Documentation ownership is limited to strict-number-resolution wording as described above.

## Files this branch must not modify

- `internal/cli/pipeline/**`
- `internal/cli/commands/lists/**`
- `internal/cli/commands/collections/**`
- `internal/cli/commands/config/**`
- `internal/cli/commands/search/**` except tests that would be strictly necessary to prove resolver behavior; prefer not to touch them
- `internal/cli/commands/magnets/**`
- `internal/cli/commands/download/**`
- root command registration for `download` / `assets`
- local-media-download documentation sections

If implementation appears to require one of these files, stop and re-evaluate the boundary rather than silently crossing it.

## Test strategy

Run focused tests after each task, then the repository default verification:

```bash
go test ./internal/javdb/appapi/endpoint/movie/...
go test ./internal/cli/commands/mark/... ./internal/cli/commands/unmark/...
go test ./sdk/...
go test ./...
sh scripts/build.sh
```

No live JavDB mutation should be required for unit tests. Use existing fake/test server patterns and make tests assert request paths/call counts when checking whether resolution was skipped.

## Acceptance criteria

- A fuzzy-only search result can no longer become a movie ID through `ResolveMovieID`.
- Exact number matching remains case-insensitive and zone-agnostic (`all`).
- `mark` batch stdin is not consumed before runner execution.
- `mark` and `unmark` use an envelope's movie ID directly.
- Existing positional `--id` behavior is preserved.
- Default test/build commands pass.
- No files owned by the other two parallel branches are modified.

## Parallel-merge contract

This branch may be developed simultaneously with:
- `fix/cli-contract-correctness`
- `refactor/download-command-clarity`

Do not cherry-pick code between these branches while development is in progress. Merge each branch independently from the same baseline. If `main` advances, rebase only after all three branches have stable tests, and resolve documentation conflicts by preserving the topic ownership rules above.

Recommended merge order is this branch first, then CLI contract correctness, then download clarity, but implementation does not depend on that order.