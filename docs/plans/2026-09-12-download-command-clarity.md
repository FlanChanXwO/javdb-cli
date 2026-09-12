# Download command clarity plan

Branch: `refactor/download-command-clarity`
Base: `main@1b21ca94a3d11008ce6028c670ea36fc2fbac3c9`

## Goal

Remove the ambiguity of the current local-media download feature across both CLI and public Go SDK without turning `javdb-cli` into a BitTorrent client, cloud-drive client, or generic download manager.

The current feature only writes selected JavDB assets (thumbnail, first preview image, preview HLS video) to local files, but both the CLI command `download` and the SDK name `DownloadMovieMedia` can reasonably be read as “download the movie/full media payload”.

Canonical CLI usage becomes:

```bash
javdb assets NUMBER --thumbnail PATH
javdb assets NUMBER --preview-image PATH
javdb assets NUMBER --preview-video PATH
```

Canonical public SDK becomes:

```go
DownloadMovieAssets(...)
MovieAssetDownloadOptions
MovieAssetDownloadResult
```

The CLI keeps `download` as a compatibility alias. The SDK performs a deliberate breaking rename and keeps **no** deprecated aliases or wrapper symbols for the old ambiguous names.

## Design decisions

### Canonical CLI name

Use `assets` as the canonical command name because the command operates on movie-associated media assets rather than full movies or torrent payloads.

Do not use `media` because it is too broad and can still imply the full video. Do not use `preview` because the command also includes the detail thumbnail. Do not use `fetch` because it hides what is being fetched.

### CLI compatibility behavior

`javdb download ...` continues to invoke the exact same local-asset behavior through Cobra alias resolution.

Requirements:
- root help and primary documentation advertise `assets`;
- `download` remains accepted as an alias;
- one Cobra command object owns both names;
- no separate legacy implementation, duplicated flags, or wrapper execution path is kept;
- no automatic full-movie/magnet download behavior is added;
- no deprecation warning is required in this first CLI migration because warnings would change stderr contracts for existing automation.

Cobra alias resolution canonicalizes the command object. Therefore `javdb download --help` may display canonical usage such as `javdb assets NUMBER` plus alias metadata. Compatibility means the old invocation is accepted and behaves identically; it does **not** require the help page to pretend `download` is still the canonical `Use` string.

### Public SDK: breaking rename, no compatibility aliases

The public Go SDK must not retain the old ambiguous exported surface after this refactor.

Rename exactly:

```go
DownloadMovieMedia        -> DownloadMovieAssets
MovieMediaDownloadOptions -> MovieAssetDownloadOptions
MovieMediaDownloadResult  -> MovieAssetDownloadResult
```

Requirements:
- remove the old exported method and exported types;
- do **not** add `type MovieMediaDownloadOptions = MovieAssetDownloadOptions`;
- do **not** add `type MovieMediaDownloadResult = MovieAssetDownloadResult`;
- do **not** add a deprecated `DownloadMovieMedia` wrapper;
- update all in-repository callers, compile-time SDK contract tests, examples and SDK documentation to the new names;
- a repository search after implementation must find no live Go reference to the three old exported identifiers outside migration/history text that intentionally names the breaking change.

This is an intentional SDK breaking change. The point of the refactor is to remove the ambiguous interface rather than preserve it indefinitely through aliases.

### Internal naming

Because `sdk/movie.go` is already being edited for the public rename, clean up directly-related private identifiers in that file where doing so reduces semantic drift:

```text
movieMediaURLsResult   -> movieAssetURLsResult
movieMediaURLs         -> movieAssetURLs
distinctMovieMediaPaths -> distinctMovieAssetPaths
```

Also prefer error text such as `movie asset output paths must be distinct` over `movie media ...` where the message describes this feature specifically.

Do not perform unrelated file moves or broad renames elsewhere merely to chase the word `media` when it is semantically correct.

### Resolver ownership boundary

This branch exclusively owns `sdk/movie.go`, `sdk/movie_test.go`, and `sdk/contract_external_test.go` so the SDK asset rename can be completed atomically.

`fix/movie-resolution-correctness` must not edit those files. It fixes public `ResolveMovieID` correctness by changing the internal endpoint that the existing SDK wrapper already calls.

Within `sdk/movie.go`, leave the `ResolveMovieID` function implementation unchanged in this branch. Do not absorb strict resolver work just because the function shares the same file.

### Pipeline protocol

Keep existing machine protocol vocabulary unchanged, including `pipeline.KindDownload` if currently used by this command.

This change deliberately cleans up human CLI naming and the public Go API. It is **not** a pipeline schema migration. Renaming `KindDownload` in the same PR would create an additional machine-contract break with little benefit to the current goal.

### Scope boundary

This branch does not implement:
- BitTorrent protocol;
- magnet selection/download;
- 115 integration;
- aria2/qBittorrent integration;
- cloud offline-download submission;
- torrent health/seed probing;
- any command or SDK API that downloads the full movie.

If a future real-download feature is wanted, it should be designed independently after this rename lands.

## Implementation tasks

### Task 1 — add failing CLI and SDK naming contract tests

Primary files:
- `internal/cli/root_test.go`
- `internal/cli/commands/download/download_test.go`
- `sdk/movie_test.go`
- `sdk/contract_external_test.go`

Before production changes, express the desired public contract:

CLI:
- root help advertises `assets` as the canonical command;
- root help no longer presents `download` as a primary command row;
- `javdb assets --help` succeeds and clearly says it saves thumbnail/preview assets only;
- `javdb download --help` succeeds through alias resolution;
- `assets` and `download` expose the same flags;
- invoking either name through the **root command tree** reaches the same execution path.

SDK:
- compile-time contract references `DownloadMovieAssets`, `MovieAssetDownloadOptions`, and `MovieAssetDownloadResult`;
- focused tests use the new option/result type names;
- initial compile/test run should fail because the new SDK symbols do not exist yet.

Go cannot conveniently write a compile-time negative assertion that an identifier no longer exists. Enforce old-symbol removal through source review/repository search in addition to successful compilation of all updated callers.

Alias behavior must be tested through the root command because Cobra aliases are resolved by the parent command. Do not create a second child wrapper solely to preserve legacy help spelling.

### Task 2 — perform the public SDK breaking rename

Primary files:
- `sdk/movie.go`
- `sdk/movie_test.go`
- `sdk/contract_external_test.go`

Rename the public types and method to the asset terminology:

```go
type MovieAssetDownloadOptions struct {
    ThumbnailPath    string
    PreviewImagePath string
    PreviewVideoPath string
}

type MovieAssetDownloadResult struct {
    // retain the current result fields and semantics
}

func (c *Client) DownloadMovieAssets(
    ctx context.Context,
    movieID string,
    opt MovieAssetDownloadOptions,
) (MovieAssetDownloadResult, error)
```

Behavior must remain identical:
- empty selection still errors;
- output paths must be distinct;
- movie detail is fetched once;
- thumbnail and preview image still use image decoding/restoration;
- preview video still downloads the complete ended HLS preview stream;
- existing-file protections remain in the underlying media layer;
- result fields and byte counts retain their current meaning.

Delete the old exported method/types rather than forwarding them to the new API.

While editing this file, rename only the directly-related private helpers listed under Internal naming. Leave unrelated movie SDK methods, including `ResolveMovieID`, behaviorally untouched.

### Task 3 — update the CLI implementation to consume the new SDK API

Primary file:
- `internal/cli/commands/download/download.go`

Replace in-repository calls/types:

```text
DownloadMovieMedia        -> DownloadMovieAssets
MovieMediaDownloadOptions -> MovieAssetDownloadOptions
MovieMediaDownloadResult  -> MovieAssetDownloadResult (where named explicitly)
```

Then make `assets` canonical:
- change `Use` to `assets NUMBER`;
- add `download` to `Aliases`;
- update `Short` / `Long` text to explicitly say “thumbnail and preview assets” and explicitly state that the command does not download full movies or magnets;
- retain all existing flags, output behavior, local-file safety checks and batch placeholders.

`internal/cli/root.go` should normally require **no production change**: it already registers the command object returned by this package, and Cobra derives the visible command name from that object's `Use`. Only edit root registration if a failing test proves it is genuinely necessary.

### Task 4 — neutralize stale resolver-specific comments without taking resolver ownership

`download.go` currently contains a comment explaining `--id` by referring to the old `ResolveMovieID` first-hit fallback. That wording becomes stale when `fix/movie-resolution-correctness` lands.

Rewrite it to stable semantics, for example:

> `--id` treats the supplied ref as the internal movie ID and bypasses printed-number resolution.

Do **not** implement or test strict resolver behavior in this branch.

### Task 5 — keep the internal package path unless a real need appears

Current package path is `internal/cli/commands/download`.

Default recommendation: **do not move the package in this PR**. The package path is internal, and moving it would add a large rename-only diff without improving the external contract.

Allowed cleanup:
- update the package comment to describe local movie assets rather than generic download behavior;
- rename directly-related local identifiers where necessary for asset terminology.

Do not rename unrelated low-level media transport APIs merely because they contain the word `Download`; `DownloadImage` and `DownloadHLS` accurately describe their operations.

### Task 6 — preserve all existing local-asset behavior

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

The intended semantic changes are naming and API discoverability, not download mechanics.

Do not opportunistically fix movie resolver semantics here; that is owned by `fix/movie-resolution-correctness`.

### Task 7 — update documentation and skill wording in owned sections

This branch owns every documentation section whose topic is local movie assets and every SDK section that names the old local-media API.

Update:
- `README.md`
- `README.zh-CN.md`
- `docs/en/cli-reference.md`
- `docs/zh-CN/cli-reference.md`
- `docs/en/sdk.md`
- `docs/zh-CN/sdk.md`
- `skills/javdb-cli/SKILL.md`
- any `skills/javdb-cli/references/*` file that instructs agents to call the current `download` command or old SDK method
- `docs/maintainers/architecture.md` only where command/API inventories name the old surface
- root/help literal expectations

Required documentation wording:
- CLI examples use `javdb assets ...`;
- explain that `download` is a CLI compatibility alias;
- SDK examples use only `DownloadMovieAssets` / `MovieAssetDownloadOptions` / `MovieAssetDownloadResult`;
- state briefly that the old SDK names were removed as a breaking clarity cleanup if a migration note is useful;
- explicitly distinguish thumbnail/preview asset writes from magnet/full-movie download;
- do not document a future downloader backend as if it already exists.

Shared-file ownership rules:
- resolver branch may edit only resolver-specific paragraphs in the same SDK/CLI docs;
- CLI-contract branch may edit only list/pipeline/config/filter paragraphs;
- this branch owns local-asset CLI/API naming paragraphs;
- do not reflow unrelated Markdown paragraphs or regenerate entire documents.

Put shared-document updates in a dedicated final documentation commit after code/tests are stable.

### Task 8 — update architecture/command inventory references without changing protocol

Where maintainers' docs list command directories or command names, clarify that the implementation package may remain `commands/download` while the canonical CLI command is `assets`.

Where SDK inventories list the old API, replace it with the new asset API.

Do not rename `pipeline.KindDownload` in this branch.

### Task 9 — verify old public SDK names are gone

After implementation and tests pass, search the repository for:

```text
DownloadMovieMedia
MovieMediaDownloadOptions
MovieMediaDownloadResult
```

Expected result:
- no Go production/test reference to the old exported symbols;
- no current API example advertises them;
- occurrences are allowed only in an intentional migration/history sentence or this plan.

This check is part of acceptance because keeping aliases would defeat the purpose of the breaking rename.

### Task 10 — full regression verification

Run focused and full checks:

```bash
go test ./sdk/...
go test ./internal/cli/commands/download/...
go test ./internal/cli/...
go test ./...
sh scripts/build.sh
```

If the project has e2e/help snapshot coverage, add or update only the cases necessary to prove `assets` canonical + `download` alias behavior.

## Files owned by this branch

Production/API ownership:
- `sdk/movie.go`
- `sdk/movie_test.go`
- `sdk/contract_external_test.go`
- `internal/cli/commands/download/**`
- `internal/cli/root_test.go` only for command-name/help expectations
- `internal/cli/root.go` only if a failing test proves registration must change; expect it to stay untouched

Documentation ownership:
- local-asset CLI sections;
- local-asset SDK sections and exported names;
- command/API inventories that name the old surface;
- skill instructions specifically describing local asset writes.

## Files this branch must not modify

- `internal/javdb/appapi/endpoint/movie/resolve.go`
- resolver tests
- `sdk/resolve_test.go` if created by the resolver branch
- `internal/cli/commands/mark/**`
- `internal/cli/commands/unmark/**`
- `internal/cli/pipeline/**`
- `internal/cli/commands/lists/**`
- `internal/cli/commands/collections/**`
- `internal/cli/commands/config/**`
- `internal/cli/commands/search/**`
- `internal/cli/commands/magnets/**`

If implementation seems to require these files, stop and re-evaluate rather than crossing branch ownership.

## Acceptance criteria

- `javdb assets NUMBER ...` is the canonical documented command.
- `javdb download NUMBER ...` remains functional as a Cobra alias and reaches the exact same implementation.
- root help lists `assets`, not `download`, as the primary command.
- alias help may canonicalize to `assets`; no wrapper command exists solely to preserve legacy help spelling.
- public SDK exposes `DownloadMovieAssets`, `MovieAssetDownloadOptions`, and `MovieAssetDownloadResult`.
- public SDK no longer exposes `DownloadMovieMedia`, `MovieMediaDownloadOptions`, or `MovieMediaDownloadResult`; no deprecated aliases/wrappers are retained.
- all in-repository callers and compile-time contract tests use the new SDK names.
- help/SDK docs make it unambiguous that only thumbnail/preview assets are written.
- stale comments in `download.go` no longer depend on the legacy fuzzy resolver behavior.
- no full movie, magnet, BitTorrent, 115, aria2, or qBittorrent behavior is introduced.
- pipeline machine kind remains unchanged.
- all existing local-asset regression behavior still passes under the canonical command and renamed SDK API.
- repository default tests/build pass.
- no production files owned by the other two parallel branches are modified.

## Parallel-merge contract

This branch is implementation-independent from:
- `fix/movie-resolution-correctness`
- `fix/cli-contract-correctness`

This branch exclusively owns `sdk/movie.go`, `sdk/movie_test.go`, and `sdk/contract_external_test.go`. The resolver branch fixes strict public number resolution below the SDK wrapper, in `internal/javdb/appapi/endpoint/movie/resolve.go`, so both branches remain independently compilable from the shared baseline.

Within `sdk/movie.go`, do not change the `ResolveMovieID` implementation. Within the resolver branch, do not touch `sdk/movie.go` at all.

Shared documentation edits must stay within the named topic anchors and be isolated in the final documentation commit.

Recommended merge order remains movie-resolution correctness -> CLI contract correctness -> download command clarity, with this branch last because it owns the final CLI/API naming surface. Never resolve a shared-doc conflict by taking one branch's whole-file version.