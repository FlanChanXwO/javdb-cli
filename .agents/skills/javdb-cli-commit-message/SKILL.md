---
name: javdb-cli-commit-message
description: Write a javdb-cli commit message from the staged diff and verified intent. Use for commit wording or preparation; staging, committing, pushing, and release actions are not implicit.
---

# Write a javdb-cli Commit Message

Read `git diff --cached --stat`, `git diff --cached`, and relevant recent subjects. If the staged diff is empty, establish the requested scope without staging anything automatically.

Write a concise English imperative subject describing the actual change, with a conventional type matching this repository. Include a body only for important motivation, compatibility, or migration effects. Do not enumerate unchanged features, invent issues, claim unrun tests, or select a release version.

Return the proposed message unless the user separately authorized committing. Preserve unrelated staged changes and keep publication a distinct action.
