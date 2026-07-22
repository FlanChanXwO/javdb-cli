# 更新日志

[English](CHANGELOG.md) | [简体中文](CHANGELOG.zh-CN.md)

本项目的重要变更记录于此。

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

## [Unreleased]

## [0.1.1] - 2026-07-23

### 新增

- `skills/javdb-cli`：包含凭据、状态变更、检索、安装与排错边界的 Agent 操作 skill。
- macOS（Intel/Apple Silicon）、Linux（amd64/arm64）与 Windows（amd64/arm64）的原生打包二进制冒烟验证。

### 变更

- 发版改为显式单目标构建/打包脚本；从不可变 tag 在全新原生 runner 重建，发布前校验资产集合，并在可选推送 tap 前验证生成的 Homebrew Formula。

### 修复

- 凭据存储权限测试现在正确处理 Windows 不公开 POSIX mode bits 的事实；支持该权限模型的平台仍强制断言 `0600`。

## [0.1.0] - 2026-07-18

首个公开版本。

### 新增

- 完整 CLI：登录、搜索、详情、磁力、标签、浏览、实体片单、用户列表、排行、TOP250、合集。
- 多账号密码登录，本地 `auth.json`，可选 `auto_relogin`。
- 公开 Go SDK 包 `javdb`。
- `javdb version --json`（供 Homebrew formula 测试）。
- 中英文 README / CONTRIBUTING / 文档。
- CI quality gate 与多架构 GitHub Release 流水线。
- Homebrew formula：`FlanChanXwO/tap/javdb-cli`。
