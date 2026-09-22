---
name: javdb-cli-release-notes
description: Prepare javdb-cli bilingual versioned changelogs with verified source attribution, coordinate an authorized release, recover downstream publishers, or synchronize historical Release bodies. Use for release work, not ordinary PR release metadata.
---

# Prepare a javdb-cli Release

Read `AGENTS.md`, `docs/maintainers/development.md`, and current release/publisher workflows. Resolve the exact version, prior tag, target commit, and authorized actions. Choosing a version does not itself permit a release-prep PR, merge, tag, publication, or history rewrite.

## Prepare and audit

1. Resolve the prior release with `sh scripts/previous-release-tag.sh vX.Y.Z`, using the approved target version.
2. Use `go run ./scripts/releasenotes audit` for the exact range and keep its report in a temporary location. Inspect current help for arguments. Squash attribution only accepts a verified PR whose `merge_commit_sha` equals the commit; a title suffix alone is not evidence.
3. Edit `changelog/vX.Y.Z/en.md` and `zh-CN.md` and their navigation. Entries carry real PR/direct-commit sources and both locales have the same source set. JavDB validates that referenced sources belong to the range; unlike Pixiv, its current policy does not require every in-range source to appear in notes.
4. If `skills/javdb-cli/` changed since the previous release, update its sole `version` to the approved SemVer and synchronize the README's published-skill version. If unchanged, preserve its prior version and verify the publisher's unchanged-skill branch. Do not bump it mechanically for every CLI release or publish maintenance skills as the product.
5. Run `sh scripts/test-releasenotes.sh` and the release-note tool's `validate` command with the version, previous tag, directory, and matching audit. Re-run audit and `validate --audit` on the exact final merged commit before creating a tag. Missing attribution, malformed notes, or API failures block tagging.

Ordinary feature PRs do not fill release-note metadata or edit versioned notes. `changelog/unreleased/` is an optional manual draft area, not an automatic workflow input.

## Publication and recovery

JavDB's release workflow accepts tag pushes and an authorized `release_tag` dispatch, both subject to source verification. Tests, production archives, verified containers and the immutable handoff precede the single `release-approval` gate. Signing and publication consume the approved identity; do not retag, rebuild different bytes under it, or bypass approval.

Homebrew, ClawHub and the container publisher run independently from a successful Release handoff. Manual recovery takes the original `release_run_id`; `publish-dockerhub.yml` publishes both GHCR and Docker Hub. Follow current workflow inputs and artifact hashes instead of treating the latest main commit as the release source. Check channel outcomes separately and preserve stable/prerelease promotion rules.

## Historical bodies

Use `validate`, `audit`, `render`, and `sync-history` through the existing tool. Dry-run each explicitly selected historical version first; only an approved `--apply` may change the remote Release body. Never create tags or replace assets as part of body synchronization. Read the result back and compare with local bilingual rendering.

Report the version, source range, Release run, actual evidence, publisher state, and remaining blockers. Do not log GitHub tokens, signing keys, account credentials, or private data. A submitted skill awaiting review is not a publicly accepted publication.
