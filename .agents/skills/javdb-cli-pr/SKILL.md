---
name: javdb-cli-pr
description: Prepare or update a javdb-cli PR and verify its diff, template, declared commands, reviews, and current-head checks. Use for contributor handoff or readiness assessment; does not authorize merging, real-service /test requests, or publication.
---

# Prepare a javdb-cli PR

Resolve the repository, base, branch, and current head. Read `AGENTS.md`, the issue/request, status, and the actual range diff. Preserve unrelated work; use [javdb-cli-test](../javdb-cli-test/SKILL.md) and [javdb-cli-review](../javdb-cli-review/SKILL.md) before handoff.

Read the target branch's `.github/PULL_REQUEST_TEMPLATE.md` and preserve its literal headings/checklist. The current parser expects the bilingual heading spelling; instructions in English do not authorize changing that wire contract. Use concise English prose under the template unless the user requests another language.

Explain changes, reasons, compatibility impact, and actual verification. Check required items honestly and identify gaps. Ordinary PRs do not require release-note metadata, a version bump, or versioned changelog entries.

A fenced `commands` block declares on-demand verification. It is not a shell transcript: inspect the trusted `tools/verification/command-whitelist.txt`, use only supported command/pipe forms, and keep credentials/private targets out of the body. Use a normal code fence for previously run evidence commands instead. Posting `/test` is an explicitly authorized live-service action, not automatic readiness bookkeeping.

For a non-secret prepared body, run from the repository root:

```bash
go run ./tools/prmeta --body-file BODY_PATH --cli javdb
```

Substitute the real body file. A passing local parser proves syntax only, not live results, remote permissions, or that the chosen targets are authorized.

Push/create/update only the authorized branch/PR. Verify checks, reviews, and mergeability on the current head SHA and use [javdb-cli-ci](../javdb-cli-ci/SKILL.md) for failures. Report passed, failed, pending, and skipped distinctly. Never force-push, merge, tag, or publish because the PR text is valid.

Include the head's commit statuses as well as check runs. Platform/Container worker runs execute under the trusted base ref; follow the smoke Check Run details URL instead of treating the absence of a PR-head workflow run as missing smoke evidence. Keep skipped checks distinct from executed passes.

Return the PR URL, commit, validation, and unresolved blockers. Local self-review does not count as a maintainer approval.
