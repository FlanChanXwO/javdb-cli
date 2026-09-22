# Discovery and Navigation

Use the main skill's preflight and verify current flags with `--help`. Identify whether the user wants a movie, person/entity, collection, ranking, or comment page before choosing a command.

## Resolve references

- `javdb search QUERY --json` can return movie, actor, series, maker, director, code, or list results. Inspect the actual kind and ID; use `--ndjson` for a typed downstream pipeline.
- `javdb detail ABC-123 --json` resolves a movie number. Add `--id` only for a verified internal movie ID. Returned entity/tag IDs can drive subsequent commands without another guessed search.
- `javdb actor|series|maker|director|code|list REF --json` reads the relevant entity's movies. `--main m --has-magnets` is a filter, not evidence of a complete catalog; use it only for the requested scope.
- Read `javdb tags --zone ZONE` before using an exact tag ID/name in `browse --tag TAG`. Tags may populate a cache on first use; `--refresh` explicitly rebuilds it and requires the requested refresh scope.

## Lists and collections

`javdb lists search QUERY --zone all --ndjson` emits `kind=list` envelopes with stable IDs and raw objects in `data.list`; `javdb list LIST_ID` reads a list's movies. Default `javdb lists` means the authenticated account's lists, not public list search. `lists related` uses a non-empty movie-envelope ID directly.

`javdb collections actors|series|codes|makers|directors --ndjson` emits one singular-kind envelope per entity, with `data.entity`. Missing stable IDs fail; missing names can fall back to IDs as references. Use explicit `--json` when the consumer needs the existing aggregate shape. Do not pass aggregate JSON to a consumer that expects envelope lines.

## Magnets, rankings, and comments

`javdb magnets ABC-123 --json` permits anonymous use and has the existing optional-token fallback. `--best` selects one preferred item (subtitles, HD, then size); omit it when the user wants all results. Apply `--cnsub`, `--hd`, or non-negative `--min-size` only for requested criteria and state those filters.

`rankings movies|actors|playback` is public; `top250` requires an account. Movie/playback results use `movies`, actor ranking uses `actors`. Movie `--type` and playback `--filter-by` accept `censored|uncensored|western|fc2`; periods are `day|week|month`. Do not substitute numeric zones or invented `daily|weekly` spellings.

`search --zone` and `lists search --zone` accept `censored|uncensored|western|fc2|all`. Search filter values are `can_play|magnets|subtitle|single`. Verify command-specific flag combinations with installed help.

`javdb comments ABC-123 --page 1 --limit 20 --json` reads that page only and keeps complete comment objects. `--id` means a verified internal ID. Do not add `--all`, fetch extra pages without scope, or call one page all comments.

At each stage, preserve errors and output completeness. Do not change keywords, accounts, proxy, or host simply to manufacture results. For preview assets, use [media.md](media.md); for a remote mark, use [state.md](state.md).
