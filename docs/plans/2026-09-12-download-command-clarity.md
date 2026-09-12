# Download command clarity plan

Branch: `refactor/download-command-clarity`
Base: `main@1b21ca94a3d11008ce6028c670ea36fc2fbac3c9`

## Goal

Remove the user-facing ambiguity of `javdb download` without turning `javdb-cli` into a BitTorrent client, cloud-drive client, or generic download manager.

The current command only writes selected JavDB media assets (thumbnail, first preview image, preview HLS video) to local files, but the public name `download` can reasonably be read as “download the movie/magnet”.

The new canonical command will be:

```bash
javdb assets NUMBER --thumbnail PATH
javdb assets NUMBER --preview-image PATH
javdb assets NUMBER --preview-video PATH
```

`download` remains a compatibility alias for this command during the migration period.

## Design decisions

### Canonical CLI name

Use `assets` as the canonical command name because the command operates on movie-associated media assets rather than full movies or torrent payloads.

Do not use `media` because it is too broad and can still imply the full video. Do not use `preview` because the command also includes the detail thumbnail. Do not use `fetch` because it hides what is being fetched.

### Compatibility behavior

`javdb download ...` must continue to invoke the exact same local-media behavior through Cobra alias resolution.

Requirements:
- root help and primary documentation advertise `assets`;
- `download` remains accepted as an alias;
- one Cobra command object owns both names;
- no separate legacy implementation, duplicated flags, or wrapper execution path is kept;
- no automatic full-movie/magnet download behavior is added;
- no deprecation warning is required in this first migration because warnings would change stderr contracts for existing automation.

Cobra alias resolution canonicalizes the command object. Therefore `javdb download --help` is allowed to display canonical usage such as `javdb assets NUMBER` plus the alias metadata. Compatibility means the old invocation is accepted and behaves identically; it does **not** require the help page to pretend `download` is still the canonical `Use` string. Tests must not force a second wrapper command merely to preserve the legacy spelling in usage text.

### Public SDK

Keep the public SDK unchanged:

```go
DownloadMovieMedia(...)
MovieMediaDownloadOptions
MovieMediaDownloadResult
```

Those names accurately describe SDK behavior and changing them would create an unrelated public Go API break.

### Pipeline protocol

Keep existing machine protocol vocabulary unchanged, including `pipeline.KindDownload` if currently used by this command.

The CLI rename is a human-facing command clarity fix. It must not silently become a pipeline schema migration.

### Scope boundary

This branch does not implement:
- BitTorrent protocol;
- magnet selection/download;
- 115 integration;
- aria2/qBittorrent integration;
- cloud offline-download submission;
- torrent health/seed probing;
- any command that downloads the full movie.

If a future real-download feature is wanted, it should be designed independently after this rename lands.

## Implementation tasks

### Task 1 — add failing command-name/help regression tests

Primary files:
- `internal/cli/root_test.go`
- `internal/cli/commands/download/download_test.go`

Before production changes, add tests that express the desired public contract:
- root help advertises `assets` as the canonical command;
- root help no longer presents `download` as a primary command row;
- `javdb assets --help` succeeds and clearly says it saves thumbnail/preview assets only;
- `javdb download --help` succeeds through alias resolution and identifies `download` as an alias/canonicalized invocation rather than requiring legacy `Use` text;
- `assets` and `download` expose the same flags;
- invoking either name through the **root command tree** reaches the same execution path.

Alias behavior must be tested through the root command, because Cobra aliases are resolved by the parent command. Do not rely only on executing the child command object directly in isolation.

The initial test run should fail against the current `download` canonical name.

### Task 2 — make `assets` canonical with `download` as alias

Primary file:
- `internal/cli/commands/download/download.go`

Expected minimal implementation:
- keep the existing command package and one Cobra command object;
- change `Use` to `assets NUMBER`;
- add `download` to `Aliases`;
- update `Short` / `Long` text to explicitly say “thumbnail and preview assets” and explicitly state that the command does not download full movies or magnets;
- retain all existing flags, output behavior, local-file safety checks and batch placeholders.

`internal/cli/root.go` should normally require **no production change**: it already registers the command object returned by the package, and Cobra derives the visible command name from that object's `Use`. Only edit root registration if a failing test proves it is genuinely necessary.

### Task 3 — neutralize stale resolver-specific comments without taking resolver ownership

`download.go` currently contains a comment explaining `--id` by referring to the old `ResolveMovieID` first-hit fallback. That wording becomes stale when `fix/movie-resolution-correctness` lands.

Because this branch owns `download.go`, rewrite such comments to stable semantics that are true both before and after the resolver branch, for example:

> `--id` treats the supplied ref as the internal movie ID and bypasses printed-number resolution.

Do **not** implement or test strict resolver behavior in this branch. This task is comment hygiene only and prevents the three branches from leaving contradictory source documentation after merge.

### Task 4 — keep the internal package path unless a real need appears

Current package path is `internal/cli/commands/download`.

Default recommendation: **do not move the package in this PR**. The package is internal and moving it would add a large rename-only diff without changing user behavior.

Allowed cleanup:
- update the package comment to describe local movie assets rather than generic download behavior;
- rename local identifiers only when directly necessary for clarity.

Do not rename SDK methods or pipeline kinds.

### Task 5 — preserve all existing media-write behavior

Regression coverage must retain current behavior for:
- `--thumbnail`;
- `--preview-image`;
- `--preview-video`;
- at least one output flag required;
- `{number}` / `{id}` placeholder expansion;
- batch targets requiring placeholders;
- duplicate target detection;
- existing-file protection;
- parent-directory validation;
- positional `--id` semantics;
- JSON/NDJSON behavior.

The only intended public semantic change is which command name is canonical.

Do not opportunistically fix movie resolver semantics here; that is owned by `fix/movie-resolution-correctness`.

### Task 6 — update documentation and skill wording in owned sections

This branch owns every documentation section whose topic is local media assets / the old `download` command name.

Update:
- `README.md`
- `README.zh-CN.md`
- `docs/en/cli-reference.md`
- `docs/zh-CN/cli-reference.md`
- `skills/javdb-cli/SKILL.md`
- any `skills/javdb-cli/references/*` file that instructs agents to call the current `download` command
- `docs/maintainers/architecture.md` only where command inventory names `download`
- root help literal expectations/tests

Required documentation wording:
- examples use `javdb assets ...`;
- explain that `download` is a compatibility alias;
- explicitly distinguish local thumbnail/preview asset writes from magnet/full-movie download;
- do not document a future downloader backend as if it already exists.

Shared-file ownership rules:
- do not edit strict movie-number resolution notes owned by `fix/movie-resolution-correctness`;
- do not edit list/pipeline/config/filter semantics owned by `fix/cli-contract-correctness`;
- do not reflow unrelated Markdown paragraphs or regenerate entire documents.

Put shared-document updates in a dedicated final documentation commit after code/tests are stable. This keeps any later merge conflict local to documentation and makes topic-preserving resolution straightforward.

### Task 7 — update architecture/command inventory references without changing protocol

Where maintainers' docs list command directories or command names, clarify that the implementation package remains `commands/download` while the canonical CLI command is `assets`.

Do not rename `pipeline.KindDownload` in this branch.

### Task 8 — full regression verification

Run focused and full checks:

```bash
go test ./internal/cli/commands/download/...
go test ./internal/cli/...
go test ./...
sh scripts/build.sh
```

If the project has e2e/help snapshot coverage, add or update only the cases necessary to prove `assets` canonical + `download` alias behavior.

## Files owned by this branch

Production ownership:
- `internal/cli/commands/download/**`
- `internal/cli/root_test.go` only for command-name/help expectations
- `internal/cli/root.go` only if a failing test proves registration must change; expect it to stay untouched

Documentation ownership:
- local-media-download / local-media-assets sections;
- command inventories that name `download`;
- skill instructions specifically describing local asset writes.

## Files this branch must not modify

- `internal/javdb/appapi/endpoint/movie/resolve.go`
- resolver tests
- `sdk/movie.go`
- `internal/cli/commands/mark/**`
- `internal/cli/commands/unmark/**`
- `internal/cli/pipeline/**`
- `internal/cli/commands/lists/**`
- `internal/cli/commands/collections/**`
- `internal/cli/commands/config/**`
- `internal/cli/commands/search/**`
- `internal/cli/commands/magnets/**`
- public SDK signatures in `sdk/`

If implementation seems to require these files, stop and re-evaluate rather than crossing branch ownership.

## Acceptance criteria

- `javdb assets NUMBER ...` is the canonical documented command.
- `javdb download NUMBER ...` remains functional as a Cobra alias and reaches the exact same implementation.
- root help lists `assets`, not `download`, as the primary command.
- alias help may canonicalize to `assets`; no wrapper command exists solely to preserve legacy help spelling.
- help text makes it unambiguous that only thumbnail/preview assets are written.
- stale comments in `download.go` no longer depend on the legacy fuzzy resolver behavior.
- no full movie, magnet, BitTorrent, 115, aria2, or qBittorrent behavior is introduced.
- SDK public names remain unchanged.
- pipeline machine kind remains unchanged.
- all existing download/media-write regression tests still pass under the canonical command.
- repository default test/build commands pass.
- no production files owned by the other two parallel branches are modified.

## Parallel-merge contract

This branch is implementation-independent from:
- `fix/movie-resolution-correctness`
- `fix/cli-contract-correctness`

It must not depend on either branch being merged before development starts. Tests for `assets` must not assert strict resolver behavior or changed pipeline/list semantics.

Production-file ownership is disjoint from the other branches. Shared documentation edits must stay within the named anchors and be isolated in the final documentation commit.

Recommended merge order remains movie-resolution correctness -> CLI contract correctness -> download command clarity, with this branch last because it owns command-inventory wording that should describe the final combined CLI. Never resolve a shared-doc conflict by taking one branch's whole-file version.