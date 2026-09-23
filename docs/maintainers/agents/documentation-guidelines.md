# Documentation Guidelines

Separate content by audience and authority rather than repeating one contract in several files.

| Content | Authoritative location |
| --- | --- |
| Installation, quick start, public overview | `README.md`, `README.zh-CN.md` |
| Public CLI and SDK contracts | `docs/en/`, with matching existing `docs/zh-CN/` pages |
| Architecture and development details | One canonical page per topic in `docs/maintainers/` |
| Documentation navigation | `docs/index.md`; old root documentation files remain link stubs |
| Contribution entry | `CONTRIBUTING.md`, `CONTRIBUTING.zh-CN.md` |
| Versioned bilingual release notes | `changelog/`; root changelog files are compatibility links |
| Always-loaded agent rules | `AGENTS.md`; client-specific files are thin pointers |
| Repository task workflows | `.agents/skills/javdb-cli-*/` |
| Installed-CLI operation | `skills/javdb-cli/` and its bundled references |

Use BCP 47 locale directories (`en`, `zh-CN`). Update the English public contract and affected existing translation in the same change. Natural rewording is welcome; different flags, safety rules, output semantics, or supported behavior are not. Route missing translations to English instead of presenting English text as a translation.

Agent contracts, maintenance/product skills, their reference files, and metadata are English. A repository maintenance workflow may link to checked-in documentation; an independently distributed product bundle cannot depend on a local checkout. Avoid personal paths, proxies, account assumptions, or globally installed skill requirements.

Keep README focused; full flags/errors/state behavior belongs in the CLI reference. SDK docs expose `sdk/` (`package javdb`), not internal integration paths. Architecture describes current ownership and flows, not reverse-engineering transcripts or credential details. Put lasting boundaries there and transient implementation reasoning in code/tests.

Synchronize documentation for changed command, API, configuration, environment, output, state, build, or test behavior. Ordinary PRs describe changes and verification; only authorized release preparation edits versioned notes and product versions. No removed release-note metadata or mandatory per-PR changelog fragment should return.

Validate links, names, metadata, and examples against actual code/help. Keep task skills small enough to read as instructions; move a conditional branch into a reference only when that cut improves navigation. Do not create a new skill for a single configuration lookup.
