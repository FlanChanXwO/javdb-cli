# CLI contract correctness plan

Branch: `fix/cli-contract-correctness`
Base: `main@1b21ca94a3d11008ce6028c670ea36fc2fbac3c9`

## Goal

Repair CLI and pipeline contract defects that can produce malformed envelopes, lose composability, ignore explicit output flags, emit misleading diagnostics, silently accept invalid filter values, or repeat authenticated fetches unnecessarily.

This branch owns **shared CLI contract correctness**. It deliberately does not own movie-number resolver semantics or the `download` command rename.

## Problems covered

1. `lists related` can receive a movie envelope with an internal ID but still resolve that ID as if it were a printed movie number.
2. `lists search` and `lists related` currently wrap arrays of lists inside a single `kind=list` envelope instead of producing one stable list envelope per result. `lists related` can even place a movie ID into a `kind=list` envelope.
3. `collections --ndjson` currently emits one aggregate envelope whose kind is derived directly from plural CLI nouns such as `actors`/`codes`. The resulting kind is invalid for most collection types, and even a singular remap alone would still be semantically wrong because one entity envelope must represent one entity, not an entire array.
4. `config get --json` / `--ndjson` with no key can be bypassed by the TTY `printAll` fast path.
5. `BatchRunner` reports the search-specific error `keyword or an image` for unrelated commands with no input.
6. `search --zone <invalid>` and `lists search --zone <invalid>` can silently omit the API zone filter instead of rejecting an invalid enum.
7. `magnets --min-size` accepts negative values and effectively disables the intended minimum-size filter.
8. `lists --json` currently performs the authenticated `MyLists` fetch twice because generic `pipeline.Producer.Execute` first calls `Produce` and then calls a separate `LegacyJSON` callback that fetches again.

## Non-goals

- Do not change `ResolveMovieID` / `ResolveMovieIDExact` or movie resolver behavior. That is owned by `fix/movie-resolution-correctness`.
- Do not modify `mark` or `unmark`.
- Do not rename `download`, add `assets`, or alter local-media-download flags. That is owned by `refactor/download-command-clarity`.
- Do not add new pipeline schema versions or new stable kind names.
- Do not change release notes or version numbers.
- Do not add new downloader/provider integrations.
- Do not globally redesign every producer just to fix `lists --json`.

## Contract decisions

### A. List fan-out

A pipeline envelope with `kind=list` represents exactly one public/personal JavDB list.

Therefore:
- `lists search QUERY --ndjson` emits zero or more `kind=list` envelopes, one per matched list;
- `lists related MOVIE --ndjson` emits zero or more `kind=list` envelopes, one per related list;
- each envelope `id` is the list ID;
- `ref` is list name with ID fallback;
- the raw API list object remains under `data.list`.

Human text and explicit legacy JSON shapes remain compatible for a single raw positional input.

### B. Envelope IDs are authoritative

For `lists related`, an input movie envelope with non-empty `id` uses that movie ID directly and must not call `ResolveMovieID`.

The `--id` flag only changes interpretation of raw positional/text input or an envelope without its own ID.

### C. Collections fan out entity envelopes

The CLI selector nouns remain plural because they select a collection:

- `actors` -> `pipeline.KindActor`
- `series` -> `pipeline.KindSeries`
- `codes` -> `pipeline.KindCode`
- `makers` -> `pipeline.KindMaker`
- `directors` -> `pipeline.KindDirector`

For pipeline execution, each collected entity becomes one envelope:
- `kind` is the singular stable protocol kind above;
- `id` and display name come from the existing `result.ProjectNamed` projection, which already normalizes `name_zht` before `name`;
- `ref` is the projected name, falling back to the projected ID;
- raw metadata is stored under `data.entity`.

Do not emit a single actor/code/etc. envelope containing the whole `items` array. Such an envelope is syntactically valid after kind normalization but still cannot be piped correctly into `actor`, `code`, `maker`, or `director` consumers.

Legacy human and single-input `--json` output remain aggregate collection views.

### D. Explicit output flags win for `config get`

`--json` and `--ndjson` always override TTY defaults.

For `config get` with no key and TTY stdin:
- no explicit output flag -> keep existing human `key=value` listing;
- `--ndjson` -> emit one valid `kind=config_key` envelope per `displayConfigKeys` entry;
- `--json` -> emit a JSON **array of those same config-key envelopes**, preserving the existing pipeline JSON contract rather than inventing a one-off raw map shape.

This deliberately matches existing cardinality semantics: `config get KEY --json` is one envelope object; `config get --json` with all display keys is multiple items, therefore an array.

Use the existing redaction path for proxy credentials. Non-TTY stdin key batches remain unchanged.

### E. Empty input error

Shared `BatchRunner` consumers report a command-specific input-required error, not search terminology.

Preferred shape:

```text
<command name>: input required
```

Use `BatchRunner.Name`; if a runner somehow has no name, fall back to `input required`. Do not add duplicated checks to every command.

Do not change the image-aware `search` command's own diagnostics unless it goes through the same runner path and a test proves the shared change is appropriate.

### F. `lists --json` must reuse the already-fetched producer result

Do **not** fix the double-fetch bug by moving `Produce` below the `OutputJSON` branch globally.

That seemingly minimal change would break `tags --refresh --json`: `tags.Produce` performs the refresh/write side effect, while its current `LegacyJSON` callback only reloads the taxonomy with refresh disabled. Skipping `Produce` in JSON mode would silently stop refresh from happening.

Preferred compatible extension:
- keep `Producer.Execute` calling `Produce` once before rendering for every mode;
- add a narrowly scoped optional callback such as `RenderJSON func(io.Writer, []Envelope) error`;
- when `RenderJSON` is present, explicit JSON serializes the already-produced envelopes and does not call `LegacyJSON`;
- when it is absent, retain the existing `Produce` -> `LegacyJSON` behavior so `tags --refresh --json` stays unchanged;
- make `lists` use `RenderJSON`, reconstructing its existing `{"lists": [...], "current_page": ...}` shape from the produced `data.list` envelopes plus the page metadata captured during `Produce`.

Do not change `tags` just to accommodate the `lists` optimization.

## Implementation tasks

### Task 1 — add failing shared-runner and producer regression tests

Primary files:
- `internal/cli/pipeline/runner_test.go`
- existing producer tests, or a focused producer test file if needed

Cover:
- missing BatchRunner input uses `Name` and no longer says `keyword or an image`;
- a producer with the new already-produced JSON renderer calls `Produce` once and that renderer once, but not `LegacyJSON`;
- a producer without the new renderer preserves the existing `Produce` then `LegacyJSON` sequence;
- text/human/NDJSON modes still call `Produce` exactly once;
- `--json` and `--ndjson` remain mutually exclusive.

The preservation test is important because it protects `tags --refresh --json` from a regression introduced by this branch.

### Task 2 — fix shared BatchRunner empty-input diagnostics

Primary file:
- `internal/cli/pipeline/runner.go`

Replace the hard-coded search-specific message with a stable command-specific input-required error derived from `Name`.

Do not add per-command callbacks unless an existing test proves one is necessary.

### Task 3 — add safe already-produced JSON rendering to `Producer`

Primary files:
- `internal/cli/pipeline/runner.go`
- producer-focused tests

Add the optional JSON renderer described in Contract F.

Important invariants:
- `Produce` still runs once before JSON rendering;
- existing producers that only define `LegacyJSON` keep their current execution order and behavior;
- no producer should fetch twice merely because it opts into the new renderer;
- `tags --refresh --json` must continue to refresh and must not be modified as part of this task.

Do not replace `LegacyJSON` globally or change its signature in this bugfix.

### Task 4 — make default `lists --json` fetch once

Primary files:
- `internal/cli/commands/lists/lists.go`
- list command tests

Keep the existing `pipeline.Producer` and existing NDJSON envelope shape (`data.list`).

During `Produce`:
- fetch `MyLists` once;
- capture `current_page` for this invocation;
- build the existing list envelopes.

Use the new producer JSON renderer to serialize those already-produced envelopes back into the existing legacy JSON shape:

```json
{"lists":[...],"current_page":"..."}
```

A fake server/client test must assert exactly one `MyLists` request for `lists --json` and unchanged JSON fields.

Do not migrate `lists` to a producer abstraction that would change `data.list` to `data.item` or otherwise mutate the pipeline schema.

### Task 5 — make `lists search` fan out one list per envelope

Primary files:
- `internal/cli/commands/lists/search.go`
- `internal/cli/commands/lists/*_test.go`

Use existing `BatchRunner.RunMany` support rather than inventing a new fan-out mechanism.

For pipeline execution:
- fetch one search result page per input query;
- return one `pipeline.KindList` envelope per list;
- set `id` from the list object's ID;
- set `ref` from list name with ID fallback;
- store the raw list under `data.list`.

Preserve the existing legacy JSON shape (`{"lists": [...]}`) and human table behavior for a single raw positional input.

### Task 6 — make `lists related` use authoritative movie IDs and fan out lists

Primary files:
- `internal/cli/commands/lists/related.go`
- related tests

Pipeline path:
- if input envelope has `id`, use it as the movie ID directly;
- otherwise resolve raw ref according to existing `--id` semantics;
- fetch related lists;
- emit one `kind=list` envelope per related list, each with the list's own ID;
- never place a movie ID in a `kind=list` envelope.

Legacy human/JSON output retains its current aggregate shape.

Do not edit movie resolver implementation files, and do not write tests that assume the strict resolver branch has already landed.

### Task 7 — make `collections` fan out valid singular entity envelopes

Primary files:
- `internal/cli/commands/collections/collections.go`
- collection command tests

Use a small explicit local mapping from plural selector -> singular pipeline kind, then use `BatchRunner.RunMany` for pipeline output.

For each fetched item:
- call the existing `result.ProjectNamed(item)` instead of duplicating entity-name normalization;
- use `row.ID` as the envelope ID;
- use `row.Name` as ref, falling back to `row.ID`;
- emit one envelope with `data.entity=item`;
- error rather than emit a success envelope if both projected name and ID are empty.

Regression tests should cover all five supported selectors and validate every emitted envelope with the existing pipeline validator. Include at least one `name_zht` case and one multi-item case proving output cardinality equals entity count.

Preserve legacy human output and single raw positional `--json` aggregate `{"items": [...]}` shape.

### Task 8 — make `config get` honor explicit output modes

Primary files:
- `internal/cli/commands/config/config.go`
- config command tests

Required cases:
- TTY + no flags + no key -> existing human all-config output;
- TTY + `--json` + no key -> JSON array of valid `config_key` envelopes for all display keys;
- TTY + `--ndjson` + no key -> one valid `config_key` envelope per display key;
- values use the same redaction behavior as keyed batch output;
- non-TTY stdin key batches remain supported;
- `--json` + `--ndjson` remains rejected.

Preferred implementation: when no key + TTY + explicit machine-output flag is requested, construct the display-key input envelopes and call the existing runner's `ExecuteWithInputs` with the resolved mode. Do not create a second bespoke JSON serializer if the runner can already provide the correct shape.

### Task 9 — reject invalid search zones in both search entry points

Primary files:
- `internal/cli/commands/search/search.go`
- `internal/cli/commands/lists/search.go`
- focused tests in both command packages

Validate CLI enum values before issuing a request. Accepted values remain exactly those advertised by help:

`censored|uncensored|western|fc2|all`.

Both commands must reject a typo before network execution. Do not change low-level SDK search behavior for arbitrary SDK callers in this branch.

A small duplicated five-value CLI check is preferable to creating a cross-package validation framework solely for two call sites.

### Task 10 — reject negative magnet minimum sizes

Primary files:
- `internal/cli/commands/magnets/magnets.go`
- magnet parser tests

`ParseSizeMiB` must return an error for negative numeric values regardless of suffix (`-1`, `-1M`, `-1GB`). Check negativity on the parsed float **before** converting to `int`, so small negative fractions cannot truncate to zero and escape validation.

Zero remains allowed and means no positive minimum filter.

Do not change ranking or `PickBestMagnet` behavior.

### Task 11 — documentation sync within this branch's owned anchors

Update only pipeline/JSON/list/filter semantics in:
- `docs/en/cli-reference.md`
- `docs/zh-CN/cli-reference.md`
- `README.md`
- `README.zh-CN.md`
- `skills/javdb-cli/SKILL.md`
- relevant skill references under `skills/javdb-cli/references/` if they describe list/collection piping or filter validation
- `docs/maintainers/architecture.md` for the optional producer JSON renderer and fan-out semantics if needed

Shared-file ownership rules:
- do not edit strict movie-number resolution paragraphs owned by `fix/movie-resolution-correctness`;
- do not edit local media download / `download` / `assets` sections owned by `refactor/download-command-clarity`;
- do not run broad Markdown reformatting or paragraph wrapping that changes unrelated hunks.

Put shared-document changes in a dedicated final documentation commit after production code/tests are stable.

## Files owned by this branch

Production ownership:
- `internal/cli/pipeline/runner.go` and focused tests
- `internal/cli/commands/lists/lists.go`
- `internal/cli/commands/lists/search.go`
- `internal/cli/commands/lists/related.go`
- `internal/cli/commands/collections/**`
- `internal/cli/commands/config/**`
- `internal/cli/commands/search/**` only for CLI zone validation
- `internal/cli/commands/magnets/**` only for `--min-size` validation

## Files this branch must not modify

- `internal/javdb/appapi/endpoint/movie/resolve.go`
- resolver tests owned by the movie-resolution branch
- `sdk/movie.go`
- `internal/cli/commands/mark/**`
- `internal/cli/commands/unmark/**`
- `internal/cli/commands/download/**`
- root registration changes for `download` / `assets`
- local-media-download documentation paragraphs
- `internal/cli/commands/tags/**` (the producer fix must preserve its behavior without touching it)

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
go test ./internal/cli/commands/tags/...   # regression guard: producer JSON changes must not break refresh behavior
go test ./...
sh scripts/build.sh
```

Where possible, validate generated envelopes with the existing pipeline validator rather than asserting only string fragments.

## Acceptance criteria

- No shared BatchRunner consumer reports `keyword or an image` for unrelated missing input.
- `lists --json` performs exactly one `MyLists` fetch and preserves its existing JSON shape.
- The producer change does not break existing `LegacyJSON` users; specifically `tags --refresh --json` still performs refresh.
- `lists search` and `lists related` emit one valid list envelope per list result.
- `lists related` never resolves an existing envelope movie ID as a printed number.
- `collections --ndjson` emits one valid singular entity envelope per collected entity, not one aggregate pseudo-entity envelope.
- `config get --json/--ndjson` wins over TTY human defaults and reuses standard config-key envelope/redaction semantics.
- invalid zones fail before network execution in both `search` and `lists search`.
- negative `--min-size` fails explicitly, including small negative fractional values.
- human and legacy JSON shapes that were already valid remain compatible.
- repository default tests/build pass.
- no production files owned by the other two branches are modified.

## Parallel-merge contract

This branch is intentionally independent from:
- `fix/movie-resolution-correctness`
- `refactor/download-command-clarity`

It may rely on the *existing interface* of `ResolveMovieID`, but must not rely on the other branch having changed its semantics yet. Tests should therefore stub resolution where needed.

Production-file ownership is disjoint from the other branches. Shared documentation edits must stay within the named anchors and be isolated in the final documentation commit.

Recommended merge order is movie-resolution correctness -> CLI contract correctness -> download command clarity, but all three can be implemented simultaneously from the shared base. When resolving later documentation conflicts, preserve topic ownership instead of choosing one branch's whole-file version.