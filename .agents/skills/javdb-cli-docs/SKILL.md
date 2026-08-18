---
name: javdb-cli-docs
description: Maintain javdb-cli documentation; locale and maintainer routing live in docs/maintainers/agents/documentation-guidelines.md.
---

# javdb-cli Docs

新增、修改或审查本仓库文档。文件职责与路由表以
`docs/maintainers/agents/documentation-guidelines.md` 为准；本文件只定义流程，不复制路由表。

## 流程

1. 读取文档规范，确定内容应落在 locale、maintainer 文档、`changelog/` 或产品 skill。
2. 按目标 locale 写作；命令、路径、包名和 code-id 保持英文。
3. 修改已翻译的 public contract 时保持行为语义对应；允许自然调整句式，不得让不同语言出现不同契约。
4. 发布说明只写入 `changelog/`：release-prep PR 直接更新目标版本的双语 notes；feature PR 不填写 release-note metadata。
5. 构建、发布、workflow 或 CI 门禁文档变化时，同步检查 `docs/maintainers/development.md`、相关 workflow 测试和 README；不要只更新用户 locale 文档。
6. 同一规则只写一处，其他位置使用链接路由；完成后检查链接、locale 导航和 `git diff --check`，并运行受影响的脚本门禁。

## 约束

- 不把长篇架构或发布说明写回 `AGENTS.md` 或根目录兼容 stub。
- 不为纯内部整理新增用户可见 changelog 条目；是否记录由 release-prep PR 根据实际用户影响决定。
- 当前项目没有 MCP server，文档不得宣称支持 MCP。
