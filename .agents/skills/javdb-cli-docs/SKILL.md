---
name: javdb-cli-docs
description: Edit javdb-cli public or maintainer documentation, repository agent contracts, maintenance skills, and the distributed operator skill. Use after changes to CLI/SDK behavior, configuration, architecture, test commands, PR policy, or publication workflows.
---

# Maintain javdb-cli Documentation

Read `AGENTS.md` and [the documentation guidelines](../../../docs/maintainers/agents/documentation-guidelines.md). Establish behavior from the relevant implementation/tests and workflow before writing it down; do not copy Pixiv-specific layers, tooling, authentication, or release semantics.

Update the actual owner: README for entry/install, locale CLI/SDK pages for public contracts, `docs/maintainers/` for engineering details, and `.agents/skills/` for task workflows. Keep root agent instructions short with precise routes. Public English/Simplified Chinese pages describe the same behavior; maintainer agent instructions, both skill trees, references, and UI metadata are English.

For CLI changes, verify arguments, number versus internal-ID semantics, output modes/cardinality, batch errors, authentication, and side effects. Preserve the asset stream's separate contract. Explain optional probe metadata and partial failures without inventing success. Read code or help before changing default-host or retry claims.

Maintenance skill names start with `javdb-cli-`; the distributed product skill remains `javdb-cli`. Keep descriptions task-specific and workflows usable through direct file reading without global skills. Product bundles must work outside a checkout: include required references and use official public URLs rather than broken repository-relative links.

Do not invent mandatory PR release-note fields or changelog fragments. Ordinary PRs state changes, verification and the current checklist; only authorized release preparation changes versioned bilingual notes and skill versions. Do not publish a skill merely because its instructions changed.

Validate metadata, local links, referenced files and commands, and English skill text. Read the result as a fresh contributor with no CCS setup. Run `git diff --check` and relevant existing script/tool checks; no separate documentation test framework is required. Root instructions are not automatically docs-only according to CI's path policy. Report the checks actually performed, not a claimed application regression run.
