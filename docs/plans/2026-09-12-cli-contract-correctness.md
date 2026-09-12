# CLI contract correctness plan

Branch: `fix/cli-contract-correctness`
Base: `main@1b21ca94a3d11008ce6028c670ea36fc2fbac3c9`

## Goal

Repair CLI and pipeline contract defects that can produce malformed envelopes, lose composability, ignore explicit output flags, emit misleading diagnostics, or silently accept invalid filter values.

This branch owns **shared CLI contract correctness**. It deliberately does not own movie-number resolver semantics or the `download` command rename.

## Problems covered

1. `lists related` can receive a movie envelope with an internal ID but still resolve that ID as if it were a printed movie number.
2. `lists search` and `lists related` currently wrap arrays of lists inside a single `kind=list` envelope instead of producing one stable list envelope per result. `lists related` can even place a movie ID into a `kind=list` envelope.
3. `collections --ndjson` converts plural CLI nouns (`actors`, `codes`, `makers`, `directors`) directly into pipeline kinds even though the protocol uses singular kinds (`actor`, `code`, `maker`, `director`).
4. `config get --json` / `--ndjson` with no key can be bypassed by the TTY `printAll` fast path.
5. `BatchRunner` reports the search-specific error `keyword or an image` for unrelated commands with no input.
6. `search --zone <invalid>` can silently omit the API zone filter instead of rejecting an invalid enum.
7. `magnets --min-size` accepts negative values and effectively disables the intended minimum-size filter.
8. `pipeline.Producer.Execute` calls `Produce` before dispatching explicit JSON to `LegacyJSON`, causing commands such as authenticated `lists --json` to perform the remote fetch twice.

## Non-goals

- Do not change `ResolveMovieID` / `ResolveMovieIDExact` or movie resolver behavior. That is owned by `fix/movie-resolution-correctness`.
- Do not modify `mark` or `unmark`.
- Do not rename `download`, add `assets`, or alter local-media-download flags. That is owned by `refactor/download-command-clarity`.
- Do not add new pipeline schema versions.
- Do not change release notes or version numbers.
- Do not add new downloader/provider integrations.

## Contract decisions

### A. List fan-out

A pipeline envelope with `kind=list` represents exactly one public/personal JavDB list.

Therefore:
- `lists search QUERY --ndjson` emits zero or more `kind=list` envelopes, one per matched list;
- `lists related MOVIE --ndjson` emits zero or more `kind=list` envelopes, one per related list;
- each envelope `id` is the list ID;
- `ref` should be the best stable human-facing list reference available, preferably list name with ID fallback;
- raw API objects may remain under `data.list` for downstream consumers that need metadata.

Human text and explicit legacy JSON shapes must remain compatible unless an existing shape is objectively malformed.

### B. Envelope IDs are authoritative

For `lists related`, an input movie envelope with non-empty `id` must use that movie ID directly and must not call `ResolveMovieID`.

The `--id` flag only changes interpretation of raw positional/text input.

### C. Collection kind mapping

CLI collection nouns remain plural because they describe collections:

- `actors` -> `pipeline.KindActor`
- `series` -> `pipeline.KindSeries`
- `codes` -> `pipeline.KindCode`
- `makers` -> `pipeline.KindMaker`
- `directors` -> `pipeline.KindDirector`

Never construct `pipeline.Kind(kind)` directly from the raw plural argument.

### D. Explicit output flags win

`--json` and `--ndjson` always override TTY defaults.

For `config get` with no key and TTY stdin:
- default mode keeps the existing human `key=value` listing;
- `--json` emits one JSON object containing all displayable config keys and values;
- `--ndjson` emits one `kind=config-key` envelope per displayable key.

For non-TTY stdin with supplied key records, existing batch semantics remain in force.

### E. Empty input error

Shared consumer commands should report a command-specific input-required error, not search terminology.

Preferred shape:

```text
<command name>: input required
```

Use `BatchRunner.Name`; do not add duplicated checks to every command.

### F. Producer JSON must fetch once

For `pipeline.Producer`, explicit JSON must not execute both `Produce` and `LegacyJSON`.

Minimal compatible approach:
- resolve output mode first;
- if mode is `OutputJSON`, invoke only `LegacyJSON`;
- otherwise invoke `Produce` and render text/NDJSON.

Do not redesign the producer API in this bugfix unless tests prove the minimal change insufficient.

## Implementation tasks

### Task 1 — add failing pipeline/producer regression tests

Primary files:
- `internal/cli/pipeline/runner_test.go`
- existing producer tests, or add a focused producer test file if none exists

Cover:
- missing input uses `BatchRunner.Name` and no longer says `keyword or an image`;
- `Producer.Execute(... --json ...)` calls `LegacyJSON` once and does not call `Produce`;
- text/human/NDJSON modes still call `Produce` once;
- `--json` and `--ndjson` remain mutually exclusive.

Observe the failures before modifying shared production code.

### Task 2 — fix shared BatchRunner empty-input diagnostics

Primary file:
- `internal/cli/pipeline/runner.go`

Replace the hard-coded search-specific message with a stable command-specific input-required error derived from `Name`.

Do not introduce per-command message callbacks unless a real existing test requires one.

### Task 3 — fix Producer explicit-JSON double fetch

Primary files:
- `internal/cli/pipeline/runner.go`
- producer-focused tests

Move the `Produce` call below the `OutputJSON` branch so `LegacyJSON` is the only producer action for explicit JSON.

Acceptance check for `lists --json`: a fake client/server should observe a single `MyLists` request.

### Task 4 — make `lists search` fan out one list per envelope

Primary files:
- `internal/cli/commands/lists/search.go`
- `internal/cli/commands/lists/*_test.go`

Use the existing `BatchRunner.RunMany` support rather than inventing a new fan-out mechanism.

For pipeline execution:
- fetch the list search result once per input query;
- return one `pipeline.KindList` envelope per result;
- set `id` from the list object's ID;
- set `ref` from list name, falling back to ID;
- store the raw item under `data.list`.

Preserve the existing legacy JSON shape (`{"lists": [...]}`) and human text table behavior for single raw positional input.

### Task 5 — make `lists related` use authoritative movie IDs and fan out lists

Primary files:
- `internal/cli/commands/lists/related.go`
- related tests

Pipeline path:
- if input envelope has `id`, use it as the movie ID directly;
- otherwise resolve raw ref according to the existing `--id` positional semantics;
- fetch related lists;
- emit one `kind=list` envelope per related list, each with the list's own ID;
- never place a movie ID in a `kind=list` envelope.

Legacy human/JSON output should retain its current aggregate shape.

Do not edit movie resolver implementation files.

### Task 6 — normalize `collections` plural nouns to protocol kinds

Primary files:
- `internal/cli/commands/collections/collections.go`
- collection command tests

Add a small explicit mapping local to the command/package. Do not modify the global pipeline `Kind` vocabulary.

Regression test each supported noun and assert emitted NDJSON validates successfully.

### Task 7 — make `config get` honor explicit output modes

Primary files:
- `internal/cli/commands/config/config.go`
- config command tests

Required cases:
- TTY + no flags + no key -> existing human all-config output;
- TTY + `--json` + no key -> valid JSON object containing all display keys;
- TTY + `--ndjson` + no key -> one valid `config-key` envelope per display key;
- non-TTY stdin key batches remain supported;
- `--json` + `--ndjson` remains rejected.

Prefer a small helper that turns `displayConfigKeys` into the existing redacted key/value representation. Do not create a generic configuration framework.

### Task 8 — reject invalid search zones

Primary files:
- `internal/cli/commands/search/search.go` and/or the narrowest existing search option validation layer
- search tests

Validate CLI enum values before issuing the request. Accepted values remain exactly those advertised by help: `censored|uncensored|western|fc2|all`.

Do not change low-level behavior for arbitrary SDK callers unless the public SDK contract explicitly promises enum validation; this task is about CLI correctness.

### Task 9 — reject negative magnet minimum sizes

Primary files:
- `internal/cli/commands/magnets/magnets.go`
- magnet parser tests

`ParseSizeMiB` must return an error for negative numeric values regardless of suffix (`-1`, `-1M`, `-1GB`). Zero remains allowed and means no positive minimum filter.

Do not change ranking or `PickBestMagnet` behavior.

### Task 10 — documentation sync within this branch's owned anchors

Update only pipeline/JSON/list/filter semantics in:
- `docs/en/cli-reference.md`
- `docs/zh-CN/cli-reference.md`
- `README.md`
- `README.zh-CN.md`
- `skills/javdb-cli/SKILL.md`
- relevant skill references under `skills/javdb-cli/references/` if they describe list piping or filter validation
- `docs/maintainers/architecture.md` only if fan-out/producer behavior needs clarification

Shared-file ownership rules:
- do not edit strict movie-number resolution paragraphs owned by `fix/movie-resolution-correctness`;
- do not edit local media download / `download` / `assets` sections owned by `refactor/download-command-clarity`;
- do not run broad Markdown reformatting or paragraph wrapping that changes unrelated hunks.

## Files owned by this branch

Production ownership:
- `internal/cli/pipeline/runner.go` and its focused tests
- `internal/cli/commands/lists/search.go`
- `internal/cli/commands/lists/related.go`
- `internal/cli/commands/collections/**`
- `internal/cli/commands/config/**`
- `internal/cli/commands/search/**` only for CLI option validation
- `internal/cli/commands/magnets/**` only for `--min-size` validation

## Files this branch must not modify

- `internal/javdb/appapi/endpoint/movie/resolve.go`
- resolver tests owned by the movie-resolution branch
- `sdk/movie.go` resolver semantics
- `internal/cli/commands/mark/**`
- `internal/cli/commands/unmark/**`
- `internal/cli/commands/download/**`
- root registration changes for `download` / `assets`
- local-media-download documentation paragraphs

If a test appears to require crossing these ownership boundaries, prefer a fake/stub around the public behavior rather than editing the other branch's files.

## Test strategy

Use test-first changes for every behavior fix.

Suggested focused commands:

```bash
go test ./internal/cli/pipeline/...
go test ./internal/cli/commands/lists/...
go test ./internal/cli/commands/collections/...
go test ./internal/cli/commands/config/...
go test ./internal/cli/commands/search/...
go test ./internal/cli/commands/magnets/...
go test ./...
sh scripts/build.sh
```

Where possible, validate generated envelopes with the existing pipeline validator rather than asserting only string fragments.

## Acceptance criteria

- No shared consumer reports `keyword or an image` for unrelated missing input.
- `Producer` explicit JSON performs one remote fetch path, not two.
- `lists search` and `lists related` emit one valid list envelope per list result.
- `lists related` never resolves an existing envelope movie ID as a printed number.
- every `collections --ndjson` supported noun emits a valid singular pipeline kind.
- `config get --json/--ndjson` wins over TTY human defaults.
- invalid search zones fail before network execution.
- negative `--min-size` fails explicitly.
- human and legacy JSON shapes that were already valid remain compatible.
- repository default tests/build pass.
- no production files owned by the other two branches are modified.

## Parallel-merge contract

This branch is intentionally independent from:
- `fix/movie-resolution-correctness`
- `refactor/download-command-clarity`

It may rely on the *existing interface* of `ResolveMovieID`, but must not rely on the other branch having changed its semantics yet. Tests should therefore stub resolution where needed.

Recommended merge order is movie-resolution correctness -> CLI contract correctness -> download command clarity, but all three can be implemented simultaneously from the shared base.

When resolving later documentation conflicts, preserve topic ownership instead of choosing one branch's whole-file version.