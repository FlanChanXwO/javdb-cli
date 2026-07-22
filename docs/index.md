# javdb-cli Documentation

`javdb-cli` is an unofficial JavDB command-line client and public Go SDK. Public
interface documents are localized; maintainer documents have one canonical
version so contributors can share the same architecture and delivery rules.

## User documentation

| Interface | English | 简体中文 |
| --- | --- | --- |
| Project overview | [README](../README.md) | [README](../README.zh-CN.md) |
| CLI reference | [English](en/cli-reference.md) | [简体中文](zh-CN/cli-reference.md) |
| Go SDK | [English](en/sdk.md) | [简体中文](zh-CN/sdk.md) |
| Contributing | [English](../CONTRIBUTING.md) | [简体中文](../CONTRIBUTING.zh-CN.md) |

English is the canonical public contract. Translations must preserve command,
flag, authentication, state-change, and error semantics; they should not be
word-for-word copies.

## Maintainer documentation

- [Architecture](maintainers/architecture.md): package boundaries, runtime flow, and ownership.
- [Development](maintainers/development.md): environment, tests, builds, and releases.
- [AI collaboration](maintainers/agents/index.md): repository instructions, review, and documentation policy.
- [Architecture decisions](maintainers/adr/): decisions that affect long-lived project boundaries.
- [Changelog](../CHANGELOG.md): user-visible changes.

Maintainer documents are currently canonical in Simplified Chinese. Add a
translation only when maintainers need to keep it current.

## Compatibility paths

The former top-level documentation paths remain as small navigation stubs for
existing links. New documentation and links must use `docs/<locale>/` or
`docs/maintainers/` directly.
