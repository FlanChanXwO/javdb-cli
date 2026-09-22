# Explicit State Changes

Require the current requested action and exact target. An explicit user instruction can supply that authorization; do not re-ask the same resolved choice. A prior operation does not authorize a different movie, account, configuration, cache, or file target.

| Action | Establish before execution |
| --- | --- |
| `mark ABC-123 --watched` | Movie, watched state, optional rating and comment |
| `mark ABC-123 --want` | Movie, wanted state, optional rating and comment |
| `unmark ABC-123` | Movie and the mark being removed |
| `auth use USER_ID` | Local account to select as default |
| `auth remove USER_ID` | Exact local account to remove |
| `config set/unset KEY` | Key, new value or reset effect |
| Cache refresh/clear | Exact cache/provider scope and intended deletion/rebuild |

`mark` requires exactly one of `--watched` and `--want`. Number resolution trims surrounding whitespace, prefers a case-insensitive full match, and accepts only an unambiguous format-equivalent match otherwise. Multiple distinct movie IDs or only fuzzy candidates fail; never choose the first result as a repair. Movie envelopes with known IDs use those directly; raw `--id` is only for a verified internal ID.

`--content` is persisted remotely. Have the user review the actual text and target; never generate and submit a comment from incidental conversation context. Treat sensitive or identifying text carefully.

Inspect exit status and report the actual outcome. Do not automatically retry an ambiguous write, change the selected account, or enable saved-password relogin to force it through.
