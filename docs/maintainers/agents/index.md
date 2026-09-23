# Agent Collaboration

The root [AGENTS.md](../../../AGENTS.md) is the entry point. It contains repository-wide boundaries and a task-to-skill table; its checked-in workflows work without a personal global contract or CCS skill collection.

## References

- [Documentation guidelines](documentation-guidelines.md): audience, locale, skill, and release-note ownership.
- [Review checklist](review-checklist.md): project-specific correctness and safety checks, used by the review skill.

## Loading and ownership

Read only the `.agents/skills/javdb-cli-*/SKILL.md` routes relevant to the task. If the client does not discover that directory, open the files directly. `CLAUDE.md` points to `AGENTS.md`; keep a single repository contract instead of restoring client-specific copies removed from the project.

Development/test skills own the Go change loop and the decision to reuse or extend existing coverage. The code-commenting skill owns documentation comments and meaningful numbered stages. Media is a specialized boundary; PR and CI handle reviewable delivery and run evidence; release notes handle explicitly authorized versions. Do not require a generic global TDD, review, commenting, or planning skill to execute any of them.

The product skill under `skills/javdb-cli/` operates an installed binary and is separately published. Keep its essential references inside the bundle and never make end users load repository-maintenance instructions. All agent instructions, skill content, and metadata are English; public translated documentation keeps its existing locale. Source comments may be English or Chinese, following the local audience and language-specific documentation syntax.
