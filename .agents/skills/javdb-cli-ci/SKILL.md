---
name: javdb-cli-ci
description: Diagnose javdb-cli quality, platform, PR-verification, and publication workflows using exact run and commit evidence. Use for failed or stalled checks, PR readiness, and authorized rerun/cancel/recovery actions; read-only investigation is the default.
---

# Diagnose javdb-cli CI

Read `AGENTS.md`, the exact workflow, its tool/script, and the current change classifier. `ci/platforms.json` and `tools/platformmatrix` own shared targets; keep workflow conditions, permissions, and executable behavior authoritative rather than copying job counts into policy documents.

Resolve repository, PR, head SHA, workflow, run ID/attempt, event, failing job/step, and first causal error through the available GitHub connector or installed `gh`:

```bash
gh pr checks PR_NUMBER
gh run view RUN_ID --json name,event,headBranch,headSha,status,conclusion,jobs,url
gh run view RUN_ID --log-failed
gh run view --job JOB_ID --log
```

Replace placeholders with verified IDs. Redact credentials and private URLs. Distinguish tool/API access errors, environment gaps, deterministic regressions, upstream failures, cancellation, and infrastructure flakiness. Slow execution alone is not a failed job.

Use [javdb-cli-test](../javdb-cli-test/SKILL.md) to reproduce the smallest relevant failure. Run safely independent downstream phases that fail-fast hides; do not repeatedly rerun unchanged deterministic failures. Honor the actual docs-only classifier and required aggregate gates rather than adding skip conditions.

`ci.yml` owns the read-only Quality check: scope classification uses trusted base-branch policy, while the Quality job executes the pull request result. Trusted `pr-metadata.yml` validates template/command declarations, and trusted `pr-smoke-gates.yml` classifies smoke scope and dispatches base-ref Platform/Container workers. Only worker matrix jobs execute the exact PR head with read-only permissions. Preserve this fork-safe boundary.

Inspect the exact PR SHA: `Quality gate`, `Platform smoke gate`, and `Container smoke gate` are GitHub Actions jobs; `PR template gate` / `PR commands gate` are commit statuses. An unnecessary Quality or smoke gate must display `Skipped`, not synthetic success. Smoke worker runs use the trusted base ref, so filtering workflow runs only by PR head can miss them; follow the smoke gate and worker runs. Ordinary branch/main pushes do not run CI; stable tag pushes run Release only, because Release already repeats the release-critical test/build verification.

PR verification treats body/commands as untrusted data. Trusted metadata/aggregation jobs must not execute PR-controlled source with write credentials; preserve the exact head and artifact identity checks. `/test` can perform real API work and requires an authorized target scope. A reaction or submitted comment is not proof that verification succeeded.

## Release boundaries

Unlike Pixiv, JavDB `release.yml` accepts both a matching tag push and an explicit `release_tag` dispatch. The existing source verifier still binds the immutable tag to the default branch. Do not transplant another repository's recovery procedure or infer a release tag from a run title.

Native archives and verified container artifacts are prepared before the single `release-approval` gate; publication consumes that identity. Homebrew, ClawHub, and `publish-dockerhub.yml` are independent downstream publishers. The latter publishes both GHCR and Docker Hub from the verified images. Their manual recovery accepts the original `release_run_id`, not a replacement tag/current-main source. Stable `latest` promotion follows the actual publisher policy.

Rerun, cancel, dispatch, push, merge, tag, or secret access requires authorization for the specific action. Do not move tags, resign/rebuild different assets, bypass review, or add retries/timeouts merely to conceal a failure. Read [javdb-cli-release-notes](../javdb-cli-release-notes/SKILL.md) for release preparation and history synchronization.

Report exact run/attempt/head, cause, observed checks, and unresolved blockers. Keep pending/skipped/failed distinct from passed; workflow files alone do not prove live protection settings or platform acceptance.
