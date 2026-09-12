# Movie resolution correctness plan

Branch: `fix/movie-resolution-correctness`
Base: `main@1b21ca94a3d11008ce6028c670ea36fc2fbac3c9`

## Goal

Make public movie-number resolution fail closed instead of silently selecting the first search hit, while preserving the existing strict candidate-search behavior already used by reverse search. Also repair the state-changing movie consumers that currently mishandle pipeline IDs or pre-read stdin.

This branch owns **movie identity correctness**. It must not absorb unrelated pipeline protocol, list fan-out, config output, magnet filter, or download-command naming work.

## User-visible problems covered

1. Public `sdk.Client.ResolveMovieID` currently reaches the legacy endpoint resolver, which prefers an exact case-insensitive number match but falls back to the first search result when no exact number is present. A mistyped number can therefore resolve to the wrong movie.
2. `mark` checks whether non-TTY stdin has content with a temporary `bufio.Reader.Peek(1)`. That reader may buffer bytes from the underlying pipe and then be discarded, so the real batch reader can observe EOF or truncated input.
3. `mark` and `unmark` use `pipeline.ConsumerRef(input)` but can still call `ResolveMovieID` when the envelope already carries an internal movie ID. In that path the internal ID is incorrectly treated as a printed number.

## Non-goals

- Do not rename or redesign `download` in this branch.
- Do not modify `lists related`; that belongs to `fix/cli-contract-correctness`.
- Do not change pipeline envelope kinds, output cardinality, JSON/NDJSON rendering, or shared empty-input diagnostics.
- Do not add BitTorrent, 115, aria2, qBittorrent, or any downloader backend.
- Do not change release notes or `changelog/vX.Y.Z/`; release-prep remains a separate process.
- Do not remove or rename `ResolveMovieIDExact`, `ResolveNumber`, or other compatibility symbols as part of this correctness fix.

## Correctness contract

### Printed movie numbers

A caller that supplies a printed number such as `SSIS-589` through public `sdk.Client.ResolveMovieID` must resolve only an exact, case-insensitive number match after trimming surrounding whitespace.

If there is no exact match, return a not-found style error. Never return the first fuzzy search hit.

If the candidate page contains multiple exact-match rows, fail rather than choose silently. Preserve the existing `ResolveNumberExact` ambiguity rule instead of inventing a second definition of exactness.

The public resolver must use the same candidate-search envelope already used by `ResolveMovieIDExact`:

- `zone=all`;
- page 1;
- limit 100;
- context-aware search via the existing `SearchContext` path.

Do **not** regress to the legacy resolver's smaller/default first page or discard the caller's context merely to reuse its function name.

`ResolveNumber` may remain as an internal compatibility helper with its historical first-hit fallback, but production `ResolveMovieID` resolution must no longer depend on it. `ResolveNumberExact` remains the source of truth for exact candidate selection.

### Pipeline movie envelopes

For an input envelope with `kind=movie` and a non-empty `id`, consumers must treat `id` as authoritative and skip number resolution. `--id` applies only to raw positional/text input or an envelope that lacks its own ID.

### `mark` stdin

`mark` must never probe stdin through a disposable buffered reader before handing the same stream to the batch runner.

The command always requires exactly one of `--watched` or `--want`. Validate that flag requirement without consuming stdin. When the status flags are valid, missing movie input remains the shared runner's responsibility; tests in this branch must not lock in the exact shared missing-input wording owned by `fix/cli-contract-correctness`.

## Implementation tasks

### Task 1 — lock down strict public resolver behavior with failing tests

Add regression tests before implementation changes.

Primary files:
- `internal/javdb/appapi/endpoint/movie/resolve_test.go`
- a focused SDK test such as `sdk/movie_test.go` if the public facade path is not already covered cleanly

Required cases:
- exact case-insensitive match succeeds;
- surrounding whitespace is accepted;
- fuzzy-only results return an error instead of the first result ID;
- no results return an error;
- multiple exact-match rows fail deterministically;
- candidate search still uses zone `all` and a limit of 100;
- public `sdk.Client.ResolveMovieID` reaches the context-aware strict path rather than the legacy first-hit resolver.

Keep the existing `ResolveNumber` compatibility test unless implementation intentionally changes that helper for a separately justified reason. The behavior under repair is public `ResolveMovieID`, not an unrelated helper symbol.

The first focused test run must demonstrate the public fuzzy-fallback failure before production code is changed.

### Task 2 — route `ResolveMovieID` through the existing strict resolver

Primary files:
- `sdk/movie.go`
- `internal/javdb/appapi/endpoint/movie/resolve.go`

Preferred implementation:
- make public `sdk.Client.ResolveMovieID(ctx, number)` delegate directly to the existing strict/context-aware `ResolveMovieIDExact(ctx, number)` capability instead of discarding `ctx`;
- make the promoted/internal `MovieEndpoint.ResolveMovieID(number)` a compatibility wrapper around the same exact resolver using `context.Background()` if keeping that method is required by existing internal adapter contracts;
- preserve all existing public function names and signatures;
- preserve `zone=all`, page 1 and limit 100 behavior;
- keep `ResolveNumberExact` as the single matching algorithm;
- do not introduce a compatibility flag for first-hit fallback.

Do not rewrite the search stack or create a repository-wide resolver abstraction. The point is to reuse the strict implementation that already exists.

### Task 3 — fix `mark` stdin consumption

Primary files:
- `internal/cli/commands/mark/mark.go`
- `internal/cli/commands/mark/mark_test.go`

Add a regression test using a non-TTY reader containing at least one movie line. The test must prove the stream reaches the batch runner intact.

Remove the disposable `bufio.Reader.Peek` pattern. Validate the mutually exclusive status flags without inspecting stdin.

Required tests:
- valid status + non-TTY stdin executes the supplied movie line;
- missing/ambiguous status fails without reading stdin;
- valid status + missing input still reaches the runner, but do not assert the exact shared error text because that belongs to the CLI-contract branch.

Do not modify shared `pipeline.BatchRunner` in this task.

### Task 4 — fix authoritative envelope-ID handling in `mark` and `unmark`

Primary files:
- `internal/cli/commands/mark/mark.go`
- `internal/cli/commands/mark/mark_test.go`
- `internal/cli/commands/unmark/unmark.go`
- `internal/cli/commands/unmark/unmark_test.go`

Required behavior:
- envelope with non-empty `id`: call the remote mutation using that ID directly;
- envelope without `id`: resolve its ref as a printed number unless `--id` explicitly says the raw ref is already an internal ID;
- positional `--id` behavior remains unchanged;
- error-envelope passthrough behavior remains owned by shared pipeline code and must not be reimplemented here.

Tests should detect accidental calls to search/resolve when an envelope ID is already present.

### Task 5 — documentation sync within this branch's owned anchors

Because resolver semantics are public SDK behavior and affect movie-number commands, update only the resolver-related wording in:
- `docs/en/sdk.md`
- `docs/zh-CN/sdk.md`
- `docs/en/cli-reference.md` and `docs/zh-CN/cli-reference.md`: only the common movie-number resolution / mutation behavior paragraph, not local-media-download or pipeline sections
- `README.md` / `README.zh-CN.md`: only a concise correctness note at an existing resolver/movie-number anchor; do not edit command inventory owned by the download branch
- `skills/javdb-cli/SKILL.md`: only the rule describing exact movie-number targeting for mutations; do not edit media-download command names

Do not reflow unrelated paragraphs. Put shared-document updates in a dedicated final documentation commit after code/tests are stable so later three-way merge conflict resolution remains mechanical.

The current `download.go` contains a comment that explicitly mentions the old first-hit resolver behavior. This branch must **not** edit that file; `refactor/download-command-clarity` owns it and its plan must neutralize that comment to wording that remains correct before and after strict resolver merge.

## Files owned by this branch

Production ownership:
- `internal/javdb/appapi/endpoint/movie/resolve.go`
- resolver tests in the same package
- `sdk/movie.go` and focused SDK resolver tests
- `internal/cli/commands/mark/**`
- `internal/cli/commands/unmark/**`

Documentation ownership is limited to strict-number-resolution wording as described above.

## Files this branch must not modify

- `internal/cli/pipeline/**`
- `internal/cli/commands/lists/**`
- `internal/cli/commands/collections/**`
- `internal/cli/commands/config/**`
- `internal/cli/commands/search/**`
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

- Public `sdk.Client.ResolveMovieID` can no longer turn a fuzzy-only search hit into a movie ID.
- Strict resolution uses the existing context-aware zone-all, limit-100 candidate search.
- Exact number matching remains case-insensitive; multiple exact rows fail closed.
- `mark` batch stdin is not consumed before runner execution.
- `mark` and `unmark` use an envelope's movie ID directly.
- Existing positional `--id` behavior is preserved.
- Existing `ResolveNumber` compatibility behavior is not accidentally broadened into public resolution.
- Default test/build commands pass.
- No files owned by the other two parallel branches are modified.

## Parallel-merge contract

This branch may be developed simultaneously with:
- `fix/cli-contract-correctness`
- `refactor/download-command-clarity`

Do not cherry-pick code between these branches while development is in progress. Production-file ownership must remain disjoint. Shared documentation edits must stay within the named anchors and be committed separately from code.

Recommended merge order is this branch first, then CLI contract correctness, then download clarity. If `main` advances, rebase after code/tests are stable and preserve all three topic-owned documentation hunks rather than taking a whole-file version from one branch.