---
name: javdb-cli-release-notes
description: Manage javdb-cli PR release-note metadata, bilingual release-prep notes, historical GitHub Release synchronization, and approved version releases.
---

# javdb-cli release notes and release workflow

在 feature PR、release-prep、merge、changelog、tag、GitHub Release 或历史 Release 正文同步时使用。
先读取 `docs/maintainers/development.md`；它是长期 policy reference。

## 授权边界

SemVer 建议不是发布授权。创建或合并 release-prep PR、创建或推送 tag、触发发布、更新 GitHub
description 或执行 `sync-history --apply` 前，必须在当前会话取得明确授权，说明目标版本、范围与
预期影响。历史同步必须列出允许更新的版本，绝不创建 tag 或替换资产。

## Feature PR

1. 检查受影响的 CLI、公开 `javdb` SDK、配置、认证、文档与产品 skill。
2. 运行 `javdb-cli-review` 和聚焦测试；修复有明确证据的 metadata、链接、格式或 policy 问题。
3. 在 PR 正文填写唯一的 `release-note` 声明：`Added`、`Changed`、`Fixed`、`Security`、`Documentation`、`Maintenance` 或 `None`；`None` 必须有理由。
4. 功能 PR 不编辑 `changelog/unreleased/`；确认 required CI 通过后再请求或等待审查。

## Release-prep

1. 审计候选范围：

   ```bash
   go run ./scripts/releasenotes audit --repo FlanChanXwO/javdb-cli --from vPREVIOUS --to COMMIT_OR_TAG --output /tmp/javdb-release-audit.json
   ```

2. 提出版本、范围、breaking assessment 与双语条目；等待明确授权。
3. 在 `changelog/plans/vX.Y.Z.json` 维护审核后的双语计划，并先预览、再显式写入：

   ```bash
   go run ./scripts/releasenotes prepare --version X.Y.Z --previous vPREVIOUS --date YYYY-MM-DD --plan changelog/plans/vX.Y.Z.json
   go run ./scripts/releasenotes prepare --version X.Y.Z --previous vPREVIOUS --date YYYY-MM-DD --plan changelog/plans/vX.Y.Z.json --apply
   go run ./scripts/releasenotes validate --version X.Y.Z --previous vPREVIOUS --dir changelog/vX.Y.Z
   ```

4. 创建 release-prep PR，运行本地门禁并等待 required CI/review。tag 只能指向已合并的 release-prep commit。

## 历史同步与发布

对每个历史版本先 dry-run，再在授权后加 `--apply`。发布后读取 GitHub Release 正文，确认它与本地
双语渲染一致。网络、GitHub API、格式或来源验证失败必须直接报告；不得伪造成功、猜测 PR 号或静默
改写其他 Release。本机联网调用显式从 `GH_TOKEN` 读取凭据；不要打印、提交或把令牌写入计划文件。
