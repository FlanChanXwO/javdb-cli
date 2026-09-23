# Review Checklist

Apply only the sections the diff touches. Every reported finding needs a concrete trigger, location, impact, and evidence; the checklist is not a mandate to invent extra layers or tests.

## Architecture and compatibility

- Keep the entry point thin and remote CLI operations on the public SDK; preserve protocol/signature ownership below it.
- Apply the local development skill's Go conventions, consumer interfaces, context/error handling, and lifetime rules.
- Preserve existing public compatibility exceptions without widening them. Verify flags, default behavior, text columns, JSON shape, NDJSON identity, cardinality, and exit status.
- Keep asset streams distinct from general pipeline envelopes. Exact identifiers must drive writes, not ambiguous search results.

## Credentials, files, and transport

- Keep passwords/JWTs/account files, signed URLs, and raw secret-bearing payloads out of logs, fixtures, and PRs.
- Require explicit account/config/cache/remote-state actions; preserve only the documented optional-auth and auto-relogin contracts.
- Verify configuration precedence and fixed/automatic host routing. Avoid guessed proxy changes or network fallbacks.
- Preserve cancellation, complete validated media, no-replace output, cleanup, and visible partial failures.
- Preserve updater signature/origin/version/platform and archive/binary hash verification before replacement; no unverified candidate execution.

## Tests and delivery

- Check actual pre-implementation Red and relevant Green/regression evidence for behavior changes, or documented exceptions. For document-only changes, validate the documents instead of inventing application tests.
- Apply the [coverage-gap decision](../../../.agents/skills/javdb-cli-test/SKILL.md#decide-whether-test-code-must-change) instead of requiring a test for each function. Select related tests, race/vet/build and script checks proportionately; honor mandatory CI and distinguish local, native, and live evidence.
- Apply the [commenting rules](../../../.agents/skills/javdb-cli-code-commenting/SKILL.md) and assess total reading effort, not comment density or function length.
- Synchronize affected public locales and product instructions. Ordinary PRs do not require versioned changelog edits or removed release metadata.
- Validate new links and task routes. Keep unrun, skipped, failed, and pending results visible; do not equate a local self-review with maintainer approval.
