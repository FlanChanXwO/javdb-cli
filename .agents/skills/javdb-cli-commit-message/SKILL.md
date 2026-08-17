---
name: javdb-cli-commit-message
description: Generate a one-line Conventional Commit message for javdb-cli from staged changes.
---

# javdb-cli Commit Message

根据暂存区生成一行提交信息。默认只读 staged changes；暂存区为空时直接说明，不编造。

## 读取

```bash
git status --short
git diff --cached
git log --oneline -10
```

## 风格

- 仅输出一行，不加解释、项目符号或代码块。
- 使用 Conventional Commits，并贴近近期风格：`feat`、`fix`、`docs`、`refactor`、`test`、`chore`、`ci`。
- subject 使用英文、小写开头，约 72 字符；不写 `misc`、`update files` 或 `wip`。
- 行为修复用 `fix`；包边界或内部结构用 `refactor`；文档和 agent 文件用 `docs`；测试用 `test`；构建、脚本、依赖与 workflow 用 `chore` 或 `ci`。
- 生成信息只描述 staged diff，不代替提交前门禁；提交前按改动范围运行 `pre-commit`、聚焦测试和项目规定的构建/文档检查，并确认工作区没有把无关改动一并提交。
