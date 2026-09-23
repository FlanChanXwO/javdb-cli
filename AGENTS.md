# javdb-cli Agent Contract

This repository provides the `javdb` CLI and the public Go SDK at `github.com/FlanChanXwO/javdb-cli/sdk` (`package javdb`).

## Start here

Read the relevant checked-in skill before its task. No personal `AGENTS.md`, CCS configuration, global skill installation, or particular agent client is required. If `.agents/skills` is not discovered automatically, open the linked `SKILL.md` directly; do not install another workflow to read it.

| Task | Local instructions |
| --- | --- |
| Implement, debug, refactor, or design Go changes | [javdb-cli-develop](.agents/skills/javdb-cli-develop/SKILL.md) |
| Select tests, run Red/Green, or validate changes | [javdb-cli-test](.agents/skills/javdb-cli-test/SKILL.md) |
| Write or review code comments, API docs, or numbered stages | [javdb-cli-code-commenting](.agents/skills/javdb-cli-code-commenting/SKILL.md) |
| Change image, HLS, or media publication behavior | [javdb-cli-media](.agents/skills/javdb-cli-media/SKILL.md) |
| Edit documentation or either kind of skill | [javdb-cli-docs](.agents/skills/javdb-cli-docs/SKILL.md) |
| Review a diff or assess a PR | [javdb-cli-review](.agents/skills/javdb-cli-review/SKILL.md) |
| Prepare, update, or verify a PR | [javdb-cli-pr](.agents/skills/javdb-cli-pr/SKILL.md) |
| Diagnose checks or operate an authorized workflow run | [javdb-cli-ci](.agents/skills/javdb-cli-ci/SKILL.md) |
| Prepare release notes or recover a publisher | [javdb-cli-release-notes](.agents/skills/javdb-cli-release-notes/SKILL.md) |
| Write a commit message from staged changes | [javdb-cli-commit-message](.agents/skills/javdb-cli-commit-message/SKILL.md) |

[The product skill](skills/javdb-cli/SKILL.md) teaches use of the installed CLI; it is separate from maintenance instructions. Use `javdb-cli-` for maintenance skill names and retain `javdb-cli` for the product skill.

## Non-negotiable boundaries

- `cmd/javdb` delegates to `internal/cli`. The CLI root assembles the command tree; `commands/<command>` owns arguments and presentation. Remote operations go through `sdk/`, never directly through App API or signature/HTTP protocol packages.
- Keep configuration and authentication lifecycle in `cli/client`, auth-store opening in `cli/authstore`, projections in `cli/result`, and reusable entity queries in `cli/entity`. The App API root is the real client composition layer, not a new forwarding facade.
- Preserve `Client.API()` and taxonomy `*tags.Doc` as existing public compatibility exceptions; do not expand the public surface with more internal types or remove these exceptions as incidental cleanup.
- Configuration precedence is CLI flags, environment, file, then defaults. CLI `host=auto` validates/reuses a cached route or explicitly reselects it; fixed hosts bypass discovery. Do not confuse this with the SDK constructor's host default.
- Preserve `javdb.pipeline/v1`, stable IDs, legacy JSON shapes, and ordered batch errors. Non-TTY default text is not NDJSON; `--ndjson` is explicit. Assets use their own `TYPE<TAB>URL` stream, not pipeline envelopes.
- `auth.json` contains passwords and JWTs. Never read real credentials into debugging output or commit credentials, downloads, caches, private URLs, or local state. Existing optional-auth anonymous retry and explicitly configured auto-relogin are intentional contracts, not permission to add new fallbacks.
- Media owns image validation, HLS decryption, TS/MP4 integrity, and no-replace publication. Save only verified complete artifacts, preserve old targets on failure, and do not advertise full-movie or magnet-target downloads.
- Preserve update signature, origin, tag, platform, archive, and binary-hash verification. The JavDB updater must not execute an unverified candidate binary. Checksums alone are not its trust root.
- Return meaningful cancellation, network, authentication, upstream, and filesystem failures. Add limits, truncation, retries, timeouts, or fallbacks only when evidence justifies them, with an explanation and regression proof.

## Working agreement

- Establish scope and acceptance evidence before editing. Use the smallest existing mechanism that meets the requirement; avoid speculative abstractions, dependencies, or symmetry with another repository. This project has no MCP server or Rust component.
- Read branch/status and preserve unrelated work. Use an isolated worktree when needed, explicit working directories, and available tools only. Prefer semantic definitions/references for code changes; disclose an unavailable LSP and verify with targeted search, compiler, and tests instead.
- Features and behavior fixes require observed Red before implementation, followed by Green and regression. Reuse or extend existing coverage before adding tests; a new function or file is not a test quota. Pure restructuring uses before/after characterization; obtain an explicit exception if an applicable Red requirement cannot be met. Ordinary comment/document edits need relevant document and tooling checks, not artificial runtime tests; consumed directives or examples may require behavioral checks.
- Use the Go version in `go.mod`, `gofmt`, and the existing test/vet/hook stack. The develop/test skills define Go design, error, concurrency, and test rules independently of global skills.
- Obtain approval before installing tools or adding dependencies. Explain necessity, alternatives, and material security, license, build, deployment, and lockfile effects; avoid incidental upgrades.
- Explain network access and material side effects. Real API tests, credential access, image uploads, persistent configuration, and state mutations need explicit authorization and targets; ordinary regression stays offline.
- Keep multi-step progress visible using a plan tool when available or a short checklist otherwise. Restore verified state before continuing; record consequential decisions in the existing issue/PR rather than generating process documents by habit.
- Keep agent instructions, both kinds of skill, skill references, and UI metadata English. Source comments may be English or Chinese: follow the local audience and [commenting rules](.agents/skills/javdb-cli-code-commenting/SKILL.md), not the language of this file. Maintain the existing public English/Simplified Chinese documentation contracts and reply in the user's requested language.
- Review meaningful changes and report exact checks, outcomes, blockers, and artifact locations. Do not translate skipped, unavailable, or pending evidence into success. A local review does not satisfy remote approval requirements.
- A PR request does not authorize a merge, release, tag change, branch-protection change, or production-secret access. Release actions require version-specific approval.

## References

[Architecture](docs/maintainers/architecture.md), [development](docs/maintainers/development.md), [agent documentation](docs/maintainers/agents/index.md), [CLI contract](docs/en/cli-reference.md), and [contributing](CONTRIBUTING.md) describe ownership and public behavior. Workflow files, `ci/platforms.json`, and manifests own executable settings; verify discrepancies instead of inferring a policy from stale prose.
