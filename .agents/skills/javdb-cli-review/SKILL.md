---
name: javdb-cli-review
description: Review javdb-cli changes with finding-first output; review criteria live in docs/maintainers/agents/review-checklist.md.
---

# javdb-cli Review

审查本仓库改动。审查标准以 `docs/maintainers/agents/review-checklist.md` 为准；本文件只定义流程和输出格式。

## 流程

1. 收集范围：`git status --short` 和 `git diff`，或用户指定的 commit/PR 范围。
2. 读取 review checklist，按架构边界、行为风险、CLI/SDK 契约、凭据与发布、测试的顺序核对。
3. 每个 finding 落到具体文件和行号；无法确认的事项列为 Open Questions，不猜测。

## 输出

```text
Findings
- [P1] path:line 问题。影响。建议修复。

Open Questions
- ...

Summary
审查范围、已运行/未运行的测试和剩余风险。
```

无 finding 时明确写“未发现阻塞问题”，并在 Summary 说明已检查范围和剩余测试风险。
