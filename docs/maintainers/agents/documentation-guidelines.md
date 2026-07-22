# 文档规范

文档按读者、语言和稳定性分层，避免把产品用法、维护者设计和 agent 指令混入同一篇根文档。

## 目录职责

- `README.md`、`README.zh-CN.md`：GitHub 项目入口和安装/快速开始。
- `docs/en/`：英文公开接口契约。
- `docs/zh-CN/`：简体中文公开接口文档。
- `docs/maintainers/`：架构、开发流程、ADR 和协作规则；每篇只保留一个 canonical 版本。
- `docs/index.md`：用户 locale 与维护者文档总导航。
- `docs/*.md`：旧链接兼容 stub，不再承载权威内容。
- `CONTRIBUTING.md` / `CONTRIBUTING.zh-CN.md`：GitHub 可发现的贡献入口。
- `CHANGELOG.md` / `CHANGELOG.zh-CN.md`：用户可感知的变化。
- `AGENTS.md`：agent 的短主规则和路由。
- `skills/javdb-cli/`：指导 agent 安全使用已安装 CLI 的产品 skill。

## Locale 规则

- locale 目录使用 BCP 47 tag：`en`、`zh-CN`。
- 先更新英文 public contract，并在同一变更中更新已有翻译；允许自然改写，不得造成不同命令、flag、安全语义或限制。
- 某语言没有真实翻译时，在 `docs/index.md` 链接到英文；不要把英文内容伪装成翻译。
- README 语言切换与文档总导航必须同步。

## 写作规则

- README 保持安装、能力边界、短示例和文档入口；完整 flag/错误/状态变更契约放 CLI reference。
- SDK 文档只描述公开 `javdb` package，不把 `internal/` 目录宣称为集成 API。
- 架构文档描述当前包边界与运行流，不记录上游逆向过程、签名推导或凭据细节。
- 长期且有取舍的决策写 ADR；短期实现细节留在代码注释和测试。
- 命令、配置、环境变量、输出语义、状态变更、构建或测试流程变化时，同步更新相应文档。
