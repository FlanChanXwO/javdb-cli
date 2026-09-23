---
name: javdb-cli-develop
description: Implement, diagnose, refactor, and design javdb-cli Go changes across CLI commands, the public SDK, App API adapters, configuration, and state. Use before source changes or engineering design; use the media workflow for image/HLS changes. Does not depend on personal agent configuration.
---

# Develop javdb-cli

## Establish the change

Read `AGENTS.md`, the relevant architecture section, exact owner code, and tests. Identify the requested result, compatibility boundary, and acceptance evidence. Reproduce bugs before selecting a fix. Resolve consequential design gaps for large work, but do not turn a clear local change into a multi-document planning exercise.

Inspect branch, base SHA, and `git status --short`. Reuse a suitable isolated worktree; otherwise use `git worktree add -b BRANCH PATH BASE` with verified values and an unused agreed path. Preserve unrelated work. Use explicit working directories and safely quoted inputs.

Use available semantic definitions/references/callers for symbol changes; disclose an unavailable LSP and inspect actual imports/callers with targeted search and compiler checks. Read exact edited code and foundational documents yourself.

## Go design and language rules

- Use the `go.mod` toolchain and `gofmt`, conventional Go initialisms, consistent receivers, and responsibility-based package names. Document affected exported contracts and state ownership, errors, cancellation, and side effects. Apply [javdb-cli-code-commenting](../javdb-cli-code-commenting/SKILL.md) to comments and numbered stages; English and Chinese prose are both acceptable, with valid Go doc syntax.
- Prefer concrete types and consumer-owned narrow interfaces at genuine substitution boundaries. Keep constructors and dependencies explicit. Do not add facade/alias/forwarder layers to imitate another repository or expose internals solely to simplify a test.
- CLI owners parse input and present output; `sdk/` owns the public capability surface; App API owns protocol/decoding. Reuse `cli/client`, `cli/result`, `cli/entity`, and `cli/pipeline` instead of copying lifecycle, projections, or stream parsing into commands.
- Treat `Client.API()` and taxonomy internal return types as frozen compatibility debt, not a precedent for new public leaks. New capabilities use typed SDK operations; preserve existing scripts' flags, output shapes, and status semantics.
- Pass context through network and media operations. Make goroutine ownership, cancellation, channels, lock scope, and cleanup explicit. Avoid shared process globals for per-invocation options and avoid locks held around uncontrolled I/O.
- Preserve typed errors with `errors.Is`/`errors.As` and useful context. Keep passwords, JWTs, signed/private URLs, and raw upstream payloads out of logs and error wrapping. Distinguish a legitimate empty result from partial failure and a failed operation.
- Keep schema validation and optional-field semantics at their real boundaries. Do not infer internal movie IDs from display names, select a fuzzy first hit for a write, or widen anonymous retry beyond the existing optional-auth contract.
- Use standard-library/platform features and installed dependencies before new code. Introduce abstractions, caches, retries, limits, and configuration only for present evidenced needs; preserve established SDK timeout/protocol policies rather than deleting them indiscriminately.

Use the official [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) for additional language guidance; apply it within the repository's existing compatibility and ownership boundaries.

## Readability and the smallest design

- Keep a coherent operation understandable in its owner. Extract a helper or package when it names a real responsibility, hides relevant complexity, or removes stable semantic duplication, not merely to shorten a function. Count the reader's cross-file jumps and parameters, not only lines removed.
- A useful interface makes its consumer simpler while its implementation owns the difficult details. Reject pass-through layers, option bags, and general frameworks whose callers still manage those details. Use SOLID as a diagnostic for actual coupling, not a requirement for one interface per type or an extension point per branch.
- Prefer explicit data flow, descriptive names, and a visible normal path over compressed expressions or clever reuse. Keep related validation and transformations near their data; remove nesting when it improves clarity without hiding distinct failures.
- Tolerate small duplication when the cases have different reasons to change. Share code only when their semantics are stable; avoid boolean modes and configuration added merely to combine unrelated cases.
- Do not build production abstractions to support oversized mocks. Select a real test boundary and the smallest fixture before changing design for test convenience. Preserve existing caller/security contracts while simplifying.

## Execute with evidence

Use [javdb-cli-test](../javdb-cli-test/SKILL.md), including its coverage-gap decision before adding tests. Reuse or extend a relevant test and observe behavioral Red before implementing a feature or fix, make the smallest change Green, then refactor with regression checks. A missing compiler, fixture, or dependency is a blocker, not a valid Red. For pure restructuring, reuse characterization before and after; any behavioral correction still needs failing evidence. Obtain an explicit exception when an applicable Red requirement cannot be run.

For image/HLS or no-replace output, also read [javdb-cli-media](../javdb-cli-media/SKILL.md). Keep input acquisition, decryption, validation, remuxing, and publication with their existing owners. Do not introduce Rust, ffmpeg, an MCP service, or a download-service layer for directory symmetry.

For new dependencies or persistent configuration, apply the root approval policy. For any new defensive constraint, name its evidence, trigger, and normal-path impact in comments and the existing delivery record; test the protected failure and valid path. Prefer fixing the underlying cause to hiding it.

Update changed public contracts through [javdb-cli-docs](../javdb-cli-docs/SKILL.md). Run available affected-file semantic diagnostics and the checks selected by the test workflow. Keep additional work tied to the acceptance criteria, not hypothetical future reuse.

## Finish

Use [javdb-cli-review](../javdb-cli-review/SKILL.md) for a meaningful change. Report exact checks and results, unverified behavior, and remaining risk. A green local check does not authorize a merge, real account operation, or release.
