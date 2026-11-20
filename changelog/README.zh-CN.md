# 更新日志

本目录按版本存放对应的英文和简体中文发布说明；GitHub Release 正文先显示英文，再显示简体中文。
章节依次为“破坏性变更、新增、变更、修复、安全、新贡献者”，省略空章节，每条附 PR 或历史直接提交来源。

正文说明用户能感知的命令、SDK 能力、安装、兼容性或安全变化，并写清升级时需要的操作。
CI、测试、内部重构、模板及维护者文档的来源放在尾部 `<!-- internal: ... -->` 注释中：
这些链接参与双语来源校验与审计，生成 GitHub Release 正文时移除。
纯内部版本仅保留版本标题、完整变更链接和内部来源；历史日期、版本归属与迁移说明保持准确。

首次提交合并 PR 的外部作者列入“新贡献者”，附 GitHub 主页和首次合并的 PR。
对照上一 tag 之前的合并历史，避免重复宣布老贡献者；省略机器人与维护者。
发布准备时根据 GitHub 贡献者名单同步两个 README 的具名头像链接，避免依赖第三方合成头像的刷新时机。

`unreleased/` 是可选人工草稿区，workflow 不读取；内容发布后清除对应草稿。
版本化说明仅在授权的发布准备或历史清理中修改。

| 版本 | 日期 | 发布说明 |
| --- | --- | --- |
| Unreleased | — | [English](unreleased/en.md) · [简体中文](unreleased/zh-CN.md) |
| [v0.8.5](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.4...v0.8.5) | 2026-11-20 | [English](v0.8.5/en.md) · [简体中文](v0.8.5/zh-CN.md) |
| [v0.8.4](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.3...v0.8.4) | 2026-11-17 | [English](v0.8.4/en.md) · [简体中文](v0.8.4/zh-CN.md) |
| [v0.8.3](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.2...v0.8.3) | 2026-10-22 | [English](v0.8.3/en.md) · [简体中文](v0.8.3/zh-CN.md) |
| [v0.8.2](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.1...v0.8.2) | 2026-09-23 | [English](v0.8.2/en.md) · [简体中文](v0.8.2/zh-CN.md) |
| [v0.8.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.8.0...v0.8.1) | 2026-09-20 | [English](v0.8.1/en.md) · [简体中文](v0.8.1/zh-CN.md) |
| [v0.8.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.3...v0.8.0) | 2026-09-20 | [English](v0.8.0/en.md) · [简体中文](v0.8.0/zh-CN.md) |
| [v0.7.3](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.2...v0.7.3) | 2026-09-08 | [English](v0.7.3/en.md) · [简体中文](v0.7.3/zh-CN.md) |
| [v0.7.2](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.1...v0.7.2) | 2026-08-18 | [English](v0.7.2/en.md) · [简体中文](v0.7.2/zh-CN.md) |
| [v0.7.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.7.0...v0.7.1) | 2026-08-14 | [English](v0.7.1/en.md) · [简体中文](v0.7.1/zh-CN.md) |
| [v0.7.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.6.1...v0.7.0) | 2026-08-14 | [English](v0.7.0/en.md) · [简体中文](v0.7.0/zh-CN.md) |
| [v0.6.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.6.0...v0.6.1) | 2026-08-13 | [English](v0.6.1/en.md) · [简体中文](v0.6.1/zh-CN.md) |
| [v0.6.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.2...v0.6.0) | 2026-08-12 | [English](v0.6.0/en.md) · [简体中文](v0.6.0/zh-CN.md) |
| [v0.5.2](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.1...v0.5.2) | 2026-08-09 | [English](v0.5.2/en.md) · [简体中文](v0.5.2/zh-CN.md) |
| [v0.5.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.5.0...v0.5.1) | 2026-08-09 | [English](v0.5.1/en.md) · [简体中文](v0.5.1/zh-CN.md) |
| [v0.5.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.4.0...v0.5.0) | 2026-08-05 | [English](v0.5.0/en.md) · [简体中文](v0.5.0/zh-CN.md) |
| [v0.4.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.3.0...v0.4.0) | 2026-08-05 | [English](v0.4.0/en.md) · [简体中文](v0.4.0/zh-CN.md) |
| [v0.3.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.2.0...v0.3.0) | 2026-08-03 | [English](v0.3.0/en.md) · [简体中文](v0.3.0/zh-CN.md) |
| [v0.2.0](https://github.com/FlanChanXwO/javdb-cli/compare/v0.1.1...v0.2.0) | 2026-07-23 | [English](v0.2.0/en.md) · [简体中文](v0.2.0/zh-CN.md) |
| [v0.1.1](https://github.com/FlanChanXwO/javdb-cli/compare/v0.1.0...v0.1.1) | 2026-07-22 | [English](v0.1.1/en.md) · [简体中文](v0.1.1/zh-CN.md) |
| [v0.1.0](https://github.com/FlanChanXwO/javdb-cli/commits/v0.1.0) | 2026-07-18 | [English](v0.1.0/en.md) · [简体中文](v0.1.0/zh-CN.md) |

格式遵循 [Keep a Changelog 1.1.0](https://keepachangelog.com/zh-CN/1.1.0/)，版本号遵循
[语义化版本](https://semver.org/lang/zh-CN/)。
