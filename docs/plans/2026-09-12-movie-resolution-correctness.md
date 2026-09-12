# Movie resolution correctness plan

Branch: `fix/movie-resolution-correctness`
Base: `main@1b21ca94a3d11008ce6028c670ea36fc2fbac3c9`

## Goal

Make public movie-number resolution fail closed instead of silently selecting the first search hit, while preserving the existing strict candidate-search behavior already used by reverse search. Also repair the state-changing movie consumers that currently mishandle pipeline IDs or pre-read stdin.

This branch owns **movie identity correctness**. It must not absorb unrelated pipeline protocol, list fan-out, config output, magnet filter, or local-asset API naming work.

## User-visible problems covered

1. Public `sdk.Client.ResolveMovieID` currently delegates to the legacy endpoint resolver, which prefers an exact case-insensitive number match but falls back to the first search result when no exact number is present. A mistyped number can therefore resolve to the wrong movie.
2. `mark` checks whether non-TTY stdin has content with a temporary `bufio.Reader.Peek(1)`. That reader may buffer bytes from the underlying pipe and then be discarded, so the real batch reader can observe EOF or truncated input.
3. `mark` and `unmark` use `pipeline.ConsumerRef(input)` but can still call `ResolveMovieID` when the envelope already carries an internal movie ID. In that path the internal ID is incorrectly treated as a printed number.

## Non-goals

- Do not rename or redesign `download` / `assets` in this branch.
- Do not rename local-asset SDK symbols; `refactor/download-command-clarity` owns that breaking API cleanup.
- Do not modify `sdk/movie.go`, `sdk/movie_test.go`, or `sdk/contract_external_test.go`; those files are owned by the download-clarity branch.
- Do not modify `lists related`; that belongs to `fix/cli-contract-correctness`.
- Do not change pipeline envelope kinds, output cardinality, JSON/NDJSON rendering, or shared empty-input diagnostics.
- Do not add BitTorrent, 115, aria2, qBittorrent, or any downloader backend.
- Do not change release notes or `changelog/vX.Y.Z/`; release-prep remains a separate process.
- Do not remove or rename `ResolveMovieIDExact`, `ResolveNumber`, or other compatibility symbols as part of this correctness fix.
- Do not expand this branch into a broader SDK context-propagation refactor. Public `ResolveMovieID` currently discards its context before delegating; this branch fixes result correctness through the endpoint it already calls without taking ownership of `sdk/movie.go`.

## Correctness contract

### Printed movie numbers

A caller that supplies a printed number such as `SSIS-589` through public `sdk.Client.ResolveMovieID` must resolve only an exact, case-insensitive number match after trimming surrounding whitespace.

If there is no exact match, return a not-found style error. Never return the first fuzzy search hit.

If the candidate page contains multiple exact-match rows, fail rather than choose silently. Preserve the existing `ResolveNumberExact` ambiguity rule instead of inventing a second definition of exactness.

The endpoint resolver used by public `sdk.Client.ResolveMovieID` must reuse the strict candidate-search behavior already implemented by `ResolveMovieIDExact`:

- `zone=all`;
- page 1;
- limit 100;
- exact matching through `ResolveNumberExact`.

`ResolveNumber` may remain as an internal compatibility helper with its historical first-hit fallback, but production `MovieEndpoint.ResolveMovieID` must no longer depend on it. `ResolveNumberExact` remains the source of truth for exact candidate selection.

Because this branch deliberately does not touch `sdk/movie.go`, it does not change the current public wrapper's context forwarding behavior. The correctness fix must therefore be achieved inside the endpoint resolver rather than by adding a competing SDK implementation.

### Pipeline movie envelopes

For an input envelope with `kind=movie` and a non-empty `id`, consumers must treat `id` as authoritative and skip number resolution. `--id` applies only to raw positional/text input or an envelope that lacks its own ID.

### `mark` stdin

`mark` must never probe stdin through a disposable buffered reader before handing the same stream to the batch runner.

The command always requires exactly one of `--watched` or `--want`. Validate that flag requirement without consuming stdin. When the status flags are valid, missing movie input remains the shared runner's responsibility; tests in this branch must not lock in the exact shared missing-input wording owned by `fix/cli-contract-correctness`.

## Implementation tasks

### Task 1 — lock down strict endpoint and public-facade behavior with failing tests

Add regression tests before implementation changes.

Primary files:
- `internal/javdb/appapi/endpoint/movie/resolve_test.go`
- new focused `sdk/resolve_test.go` if a public SDK regression test is needed

Do **not** put resolver tests in `sdk/movie_test.go`; the download-clarity branch owns that existing file for the SDK asset rename.

Required cases:
- exact case-insensitive match succeeds;
- surrounding whitespace is accepted;
- fuzzy-only results return an error instead of the first result ID;
- no results return an error;
- multiple exact-match rows fail deterministically;
- endpoint candidate search uses zone `all` and limit 100;
- public `sdk.Client.ResolveMovieID` becomes strict through its existing `c.api.ResolveMovieID(number)` delegation path.

Keep the existing `ResolveNumber` compatibility test. The behavior under repair is `MovieEndpoint.ResolveMovieID` and therefore public `sdk.Client.ResolveMovieID`, not the legacy helper itself.

The first focused test run must demonstrate the fuzzy-fallback failure before production code is changed.

### Task 2 — make the existing endpoint resolver delegate to strict resolution

Primary file:
- `internal/javdb/appapi/endpoint/movie/resolve.go`

Preferred implementation:
- keep the existing `MovieEndpoint.ResolveMovieID(number string)` signature so the public SDK wrapper requires no edit;
- implement it as a thin compatibility entry point to the already-existing strict resolver, e.g. `ResolveMovieIDExact(context.Background(), number)`;
- thereby reuse `SearchContext`, `zone=all`, page 1, limit 100, and `ResolveNumberExact` rather than maintaining two algorithms;
- preserve `ResolveMovieIDExact(ctx, number)` for callers that already pass a context;
- preserve `ResolveNumber` as a legacy helper, but stop using it from production `ResolveMovieID` resolution;
- do not introduce a compatibility flag for first-hit fallback.

Do not rewrite the search stack or create a repository-wide resolver abstraction. The point is to collapse production number resolution onto the strict implementation while leaving the SDK file available exclusively to the parallel asset-rename branch.

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

Because resolver semantics affect movie-number targeting, update only resolver-related wording in:
- `docs/en/sdk.md`
- `docs/zh-CN/sdk.md`
- `docs/en/cli-reference.md` and `docs/zh-CN/cli-reference.md`: only the common movie-number resolution / mutation behavior paragraph, not local-asset or pipeline sections
- `README.md` / `README.zh-CN.md`: only a concise correctness note at an existing resolver/movie-number anchor; do not edit command inventory owned by the download branch
- `skills/javdb-cli/SKILL.md`: only the rule describing exact movie-number targeting for mutations; do not edit local-asset command or SDK names

Do not reflow unrelated paragraphs. Put shared-document updates in a dedicated final documentation commit after code/tests are stable so later three-way merge conflict resolution remains mechanical.

The current `download.go` contains a comment that explicitly mentions the old first-hit resolver behavior. This branch must **not** edit that file; `refactor/download-command-clarity` owns it and its plan must neutralize that comment to wording that remains correct before and after strict resolver merge.

## Files owned by this branch

Production ownership:
- `internal/javdb/appapi/endpoint/movie/resolve.go`
- resolver tests in the same package
- new `sdk/resolve_test.go` only if public-facade coverage is needed
- `internal/cli/commands/mark/**`
- `internal/cli/commands/unmark/**`

Documentation ownership is limited to strict-number-resolution wording as described above.

## Files this branch must not modify

- `sdk/movie.go`
- `sdk/movie_test.go`
- `sdk/contract_external_test.go`
- `internal/cli/pipeline/**`
- `internal/cli/commands/lists/**`
- `internal/cli/commands/collections/**`
- `internal/cli/commands/config/**`
- `internal/cli/commands/search/**`
- `internal/cli/commands/magnets/**`
- `internal/cli/commands/download/**`
- root command registration for `download` / `assets`
- local-asset SDK documentation sections and exported names

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

- Public `sdk.Client.ResolveMovieID` can no longer turn a fuzzy-only search hit into a movie ID through its existing endpoint delegation.
- Production number resolution uses the existing strict zone-all, limit-100 candidate search.
- Exact number matching remains case-insensitive; multiple exact rows fail closed.
- `mark` batch stdin is not consumed before runner execution.
- `mark` and `unmark` use an envelope's movie ID directly.
- Existing positional `--id` behavior is preserved.
- Existing `ResolveNumber` compatibility behavior is not accidentally broadened into production resolution.
- `sdk/movie.go` and SDK asset symbols are untouched by this branch.
- Default test/build commands pass.
- No files owned by the other two parallel branches are modified.

## Parallel-merge contract

This branch may be developed simultaneously with:
- `fix/cli-contract-correctness`
- `refactor/download-command-clarity`

`refactor/download-command-clarity` exclusively owns `sdk/movie.go`, `sdk/movie_test.go`, and `sdk/contract_external_test.go` for the SDK asset rename. This branch must obtain strict public resolution solely by changing the endpoint already called by the unchanged SDK wrapper.

Do not cherry-pick code between these branches while development is in progress. Production-file ownership must remain disjoint. Shared documentation edits must stay within the named anchors and be committed separately from code.

Recommended merge order is this branch first, then CLI contract correctness, then download clarity. If `main` advances, rebase after code/tests are stable and preserve all three topic-owned documentation hunks rather than taking a whole-file version from one branch.