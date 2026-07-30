## Summary

<!-- Describe the user problem and the change that addresses it. Link related issues with "Fixes #123" when applicable. -->

## Scope and compatibility

<!-- List affected CLI commands or flags, public Go SDK APIs, configuration, environment variables, output contracts, agent skill files, and release behavior. State "None" when there is no public impact. Do not claim MCP support: javdb-cli has no MCP server. -->

## Release note metadata

<!--
This required metadata is validated by CI and is used only when a later release-prep PR writes bilingual versioned notes.
Do not edit changelog/unreleased in a feature PR. Choose exactly one category: Added, Changed, Fixed, Security,
Documentation, Maintenance, or None. Set breaking to true only when the change needs a major-version release. None
requires a concrete reason.
-->

<!-- release-note
category: None
breaking: false
summary: Explain the user-visible outcome in one English sentence.
none_reason: Explain why this pull request has no user-visible release note.
-->

## Verification

<!-- List the exact commands you ran and their results. For real JavDB App API coverage, state whether it was run and use only redacted evidence. -->

```text
go test ./...
sh scripts/build.sh
```

## Checklist

- [ ] The change is focused and linked to an issue when appropriate.
- [ ] I added or updated focused tests for changed behavior.
- [ ] I ran the relevant tests and recorded the results above.
- [ ] I updated the required CLI reference, SDK, README, maintainer, and operator-skill documentation.
- [ ] I completed the required release-note declaration above; `None` has a concrete reason when selected.
- [ ] I documented every new timeout, retry, pagination or result limit, truncation, fallback, or downgrade and its evidence.
- [ ] I did not add passwords, JWTs, `~/.javdb-cli/auth.json`, proxy credentials, private URLs, local state, or private API responses.
- [ ] I updated migration guidance for every breaking change.
