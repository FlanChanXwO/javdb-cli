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

`javdb download ...` must continue to invoke the same local-media behavior as an alias.

Requirements:
- root help and primary documentation advertise `assets`;
- `download` remains accepted by Cobra as an alias;
- no separate legacy implementation is kept;
- no duplicated flags or execution path;
- no automatic full-movie/magnet download behavior is added;
- no deprecation warning is required in this first migration because warnings would change stderr contracts for existing automation.

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
- `internal/cli/commands/download/download_test.go` or the eventual renamed package test file

Before production changes, add tests that express the desired public contract:
- root help advertises `assets` as the canonical command;
- root help no longer presents `download` as the primary command row;
- `javdb assets --help` succeeds and clearly says it saves thumbnail/preview assets only;
- `javdb download --help` still succeeds through the compatibility alias;
- `assets` and `download` expose the same flags;
- invoking either name reaches the same execution path.

The initial test run should fail against the current `download` canonical name.

### Task 2 — make `assets` canonical with `download` as alias

Primary files:
- `internal/cli/commands/download/download.go`
- `internal/cli/root.go`

Preferred minimal implementation:
- keep one Cobra command object;
- change `Use` to `assets NUMBER`;
- add `download` as a Cobra alias;
- update `Short` / `Long` text to explicitly say “thumbnail and preview assets” and explicitly state that the command does not download full movies or magnets;
- retain all existing flags and behavior;
- retain existing local-file safety checks and batch placeholders.

Avoid a second wrapper command that delegates to `assets`; aliases should share one implementation.

### Task 3 — decide whether to rename the internal package only if it improves clarity without churn

Current package path is `internal/cli/commands/download`.

Default recommendation: **do not move the package in this PR**. The package is internal and moving it would add a large rename-only diff without changing user behavior.

Allowed cleanup:
- update the package comment to describe local movie assets rather than generic download behavior;
- rename internal local identifiers only when directly necessary for clarity.

Do not rename SDK methods or pipeline kinds.

### Task 4 — preserve all existing media-write behavior

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

### Task 5 — update documentation and skill wording in owned sections

This branch owns every documentation section whose topic is local media assets / the old `download` command name.

Update:
- `README.md`
- `README.zh-CN.md`
- `docs/en/cli-reference.md`
- `docs/zh-CN/cli-reference.md`
- `skills/javdb-cli/SKILL.md`
- any `skills/javdb-cli/references/*` file that instructs agents to call the current `download` command
- `docs/maintainers/architecture.md` only where the command inventory names `download`
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

### Task 6 — update architecture/command inventory references without changing protocol

Where maintainers' docs list command directories or command names, clarify that the implementation package may remain `commands/download` while the canonical CLI command is `assets`.

Do not rename `pipeline.KindDownload` in this branch.

### Task 7 — full regression verification

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
- `internal/cli/root.go` only for registration/import/help effects related to this command
- `internal/cli/root_test.go` only for command-name/help expectations

Documentation ownership:
- local-media-download / local-media-assets sections;
- command inventories that name `download`;
- skill instructions specifically describing local asset writes.

## Files this branch must not modify

- `internal/javdb/appapi/endpoint/movie/resolve.go`
- resolver tests
- `internal/cli/commands/mark/**`
- `internal/cli/commands/unmark/**`
- `internal/cli/pipeline/runner.go`
- `internal/cli/commands/lists/**`
- `internal/cli/commands/collections/**`
- `internal/cli/commands/config/**`
- `internal/cli/commands/search/**`
- `internal/cli/commands/magnets/**`
- public SDK signatures in `sdk/` unless a comment-only clarification is strictly required; prefer docs instead

If implementation seems to require these files, stop and re-evaluate rather than crossing branch ownership.

## Acceptance criteria

- `javdb assets NUMBER ...` is the canonical documented command.
- `javdb download NUMBER ...` remains functional as an alias.
- help text makes it unambiguous that only thumbnail/preview assets are written.
- no full movie, magnet, BitTorrent, 115, aria2, or qBittorrent behavior is introduced.
- SDK public names remain unchanged.
- pipeline machine kind remains unchanged.
- all existing download/media-write regression tests still pass under the canonical command.
- repository default test/build commands pass.
- no production files owned by the other two parallel branches are modified.

## Parallel-merge contract

This branch is independent from:
- `fix/movie-resolution-correctness`
- `fix/cli-contract-correctness`

It must not depend on either branch being merged first. In particular, tests for `assets` should not assert the new strict resolver behavior; resolver correctness belongs to the other branch.

Recommended merge order is movie-resolution correctness -> CLI contract correctness -> download command clarity, but all three branches are safe to implement simultaneously from the shared base.

For shared documentation files, merge by topic ownership. Never resolve a conflict by taking one branch's whole-file version, because that could discard another branch's independently valid documentation updates.