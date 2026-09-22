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

- Use the `go.mod` toolchain and `gofmt`, conventional Go initialisms, consistent receivers, and responsibility-based package names. Write exported API documentation in English and state ownership, errors, cancellation, and side effects. Keep comments about intent and constraints, not restatements of syntax.
- Prefer concrete types and consumer-owned narrow interfaces at genuine substitution boundaries. Keep constructors and dependencies explicit. Do not add facade/alias/forwarder layers to imitate another repository or expose internals solely to simplify a test.
- CLI owners parse input and present output; `sdk/` owns the public capability surface; App API owns protocol/decoding. Reuse `cli/client`, `cli/result`, `cli/entity`, and `cli/pipeline` instead of copying lifecycle, projections, or stream parsing into commands.
- Treat `Client.API()` and taxonomy internal return types as frozen compatibility debt, not a precedent for new public leaks. New capabilities use typed SDK operations; preserve existing scripts' flags, output shapes, and status semantics.
- Pass context through network and media operations. Make goroutine ownership, cancellation, channels, lock scope, and cleanup explicit. Avoid shared process globals for per-invocation options and avoid locks held around uncontrolled I/O.
- Preserve typed errors with `errors.Is`/`errors.As` and useful context. Keep passwords, JWTs, signed/private URLs, and raw upstream payloads out of logs and error wrapping. Distinguish a legitimate empty result from partial failure and a failed operation.
- Keep schema validation and optional-field semantics at their real boundaries. Do not infer internal movie IDs from display names, select a fuzzy first hit for a write, or widen anonymous retry beyond the existing optional-auth contract.
- Use standard-library/platform features and installed dependencies before new code. Introduce abstractions, caches, retries, limits, and configuration only for present evidenced needs; preserve established SDK timeout/protocol policies rather than deleting them indiscriminately.

Use the official [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments) for additional language guidance; apply it within the repository's existing compatibility and ownership boundaries.

## Execute with evidence

Use [javdb-cli-test](../javdb-cli-test/SKILL.md). Observe a relevant behavioral Red before implementing a feature or fix, make the smallest change Green, then refactor with regression checks. A missing compiler, fixture, or dependency is a blocker, not a valid Red. For pure restructuring, characterize and verify behavior before and after; any behavioral correction still needs its own failing test. Obtain an explicit exception when the required Red cannot be run.

For image/HLS or no-replace output, also read [javdb-cli-media](../javdb-cli-media/SKILL.md). Keep input acquisition, decryption, validation, remuxing, and publication with their existing owners. Do not introduce Rust, ffmpeg, an MCP service, or a download-service layer for directory symmetry.

For new dependencies or persistent configuration, apply the root approval policy. For any new defensive constraint, name its evidence, trigger, and normal-path impact in comments and the existing delivery record; test the protected failure and valid path. Prefer fixing the underlying cause to hiding it.

Update changed public contracts through [javdb-cli-docs](../javdb-cli-docs/SKILL.md). Run available affected-file semantic diagnostics and the checks selected by the test workflow. Keep additional work tied to the acceptance criteria, not hypothetical future reuse.

## Finish

Use [javdb-cli-review](../javdb-cli-review/SKILL.md) for a meaningful change. Report exact checks and results, unverified behavior, and remaining risk. A green local check does not authorize a merge, real account operation, or release.
