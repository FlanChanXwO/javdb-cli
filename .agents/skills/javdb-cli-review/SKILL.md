---
name: javdb-cli-review
description: Review javdb-cli diffs or PRs for behavior, Go design, public SDK compatibility, pipeline/media correctness, credential safety, and verification gaps. Use for independent or pre-delivery review; changes and external actions need separate authorization.
---

# Review javdb-cli

Inspect status, diff statistics, and the full relevant diff before broader navigation. Resolve the exact base/head for PRs and committed work; distinguish it from the working-tree diff. Read the requirement, `AGENTS.md`, and [the review checklist](../../../docs/maintainers/agents/review-checklist.md).

Trace the real flow and confirm the requested behavior. Check the Go design rules in [javdb-cli-develop](../javdb-cli-develop/SKILL.md), SDK boundaries, context/error propagation, stream identity/cardinality, auth modes, persistence, and no-overwrite behavior. Review media changes with [javdb-cli-media](../javdb-cli-media/SKILL.md).

Pay particular attention to default piped text versus explicit NDJSON, legacy JSON projections, exact movie resolution before writes, optional-auth anonymous retry, auto-route caching, redacted credentials, and update trust before replacing a binary. Existing compatibility exceptions are neither permission to widen them nor permission to remove them silently.

Check test evidence and documentation against the actual change. Do not demand release metadata removed from the PR template or a new changelog entry for every feature PR. Review workflow security and artifact trust when affected; avoid assertions tied only to YAML wording, step counts, or a preference for more layers.

Apply the [coverage-gap decision](../javdb-cli-test/SKILL.md#decide-whether-test-code-must-change) before requesting another test: identify the missing scenario and why existing checks would miss it, not a newly introduced function or a desired test count. Apply [the commenting rules](../javdb-cli-code-commenting/SKILL.md) to API contracts and workflow stages. Judge readability by total mental effort, including cross-file jumps; neither shorter functions nor more comments prove improvement. Either source-comment language is acceptable, while directives and factual guarantees remain exact.

Verify each material finding at an exact location or with a focused safe reproduction. Report severity, `file:line`, trigger, impact, evidence, and the smallest correct fix. Distinguish proven defects from uncertainty. If clear, state the reviewed scope and unverified behavior rather than an unconditional guarantee.

Remain review-only unless editing is authorized. Self-review does not satisfy remote reviewer approval, and a green subset of checks does not authorize merging or publishing.
