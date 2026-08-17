---
name: javdb-cli-review
description: Review javdb-cli changes with finding-first output; review criteria live in docs/maintainers/agents/review-checklist.md.
---

# javdb-cli Review

审查本仓库改动。审查标准以 `docs/maintainers/agents/review-checklist.md` 为准；本文件只定义流程和输出格式。

## 流程

1. 收集范围：若目标是 PR，先用 `gh pr view NUMBER --json baseRefName,headRefName,baseRefOid,headRefOid,state,isDraft,mergeable,mergeStateStatus,statusCheckRollup,reviewDecision,body,url` 和 `gh pr diff NUMBER` 固定远端 base/head SHA；不要让脏工作区的 `git diff` 替代 PR diff。只有用户明确要审查本地改动时才使用 `git status --short`/`git diff`。
2. 读取 review checklist，按架构边界、行为风险、CLI/SDK 契约、凭据与发布、测试的顺序核对。
3. PR 范围还要检查模板中的 release-note 四字段、required CI、mergeability、review/approval 状态、目标分支保护和是否存在 bypass 依赖；这些属于发布前置条件，不是代码 finding。
4. 每个代码 finding 落到具体文件和行号；CI、PR 状态或策略问题用对应的 URL、job/check 名称和 SHA 定位；无法确认的事项列为 Open Questions，不猜测。

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
