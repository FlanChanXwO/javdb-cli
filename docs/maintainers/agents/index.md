# AI 协作文档

本目录承接 `AGENTS.md` 不适合长期展开的协作细则。根指令保持短且可执行；
按任务需要再读取这里的专项规则。

## 文档地图

- [Review checklist](review-checklist.md)：代码、接口、文档和发布变更的审查清单。
- [Documentation guidelines](documentation-guidelines.md)：README、locale、维护者文档、ADR、skills 与 changelog 的边界。

## 使用原则

- `AGENTS.md` 是仓库内 agent 的主规则与目录路由。
- `CLAUDE.md` 只引用 `AGENTS.md`，不维护第二份规则。
- `.github/copilot-instructions.md` 是短提示，不复制主规则全文。
- `skills/javdb-cli/` 是面向产品使用者的 skill；它不是仓库重构、审查或发布规则的替代品。
