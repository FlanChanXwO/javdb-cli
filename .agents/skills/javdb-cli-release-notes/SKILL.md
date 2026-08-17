---
name: javdb-cli-release-notes
description: Manage javdb-cli PR release-note metadata, bilingual release-prep notes, squash-merge source attribution, historical GitHub Release synchronization, and approved version releases. Use this whenever a user mentions release notes, release-prep, merge, tag, GitHub Release, CI release checks, or a release retry.
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

1. 先确定发布边界：上一稳定 tag、最终要发布的 `main` commit，以及本次 release-prep PR 自身的 PR URL。不要用脏工作区、未合并 branch tip 或猜测的 PR 号作为最终来源。

2. 在 release-prep PR 合并后、创建 tag 前，对最终 `main` commit 做一次精确审计：

   ```bash
   go run ./scripts/releasenotes audit --repo FlanChanXwO/javdb-cli --from vPREVIOUS --to FINAL_MAIN_COMMIT --output /tmp/javdb-release-audit.json
   go run ./scripts/releasenotes validate --version X.Y.Z --previous vPREVIOUS --dir changelog/vX.Y.Z --audit /tmp/javdb-release-audit.json
   ```

   只有 audit report 的 PR/commit 来源集合与 release plan、双语 notes 完全一致时才能继续。审计失败、GitHub API 来源查询失败或出现未归因 direct commit 时停止，不创建 tag。

   GitHub 的 squash merge 可能不会让 `/commits/{sha}/pulls` 返回 PR。审计工具只接受提交标题末尾的 GitHub `(#N)` 后缀，并再次确认该 PR 的 `merge_commit_sha` 精确等于提交 SHA；不能靠标题猜测或手工伪造来源。

3. 提出版本、范围、breaking assessment 与双语条目；等待明确授权。
4. 在 `changelog/plans/vX.Y.Z.json` 维护审核后的双语计划，并先预览、再显式写入：

   ```bash
   go run ./scripts/releasenotes prepare --version X.Y.Z --previous vPREVIOUS --date YYYY-MM-DD --plan changelog/plans/vX.Y.Z.json
   go run ./scripts/releasenotes prepare --version X.Y.Z --previous vPREVIOUS --date YYYY-MM-DD --plan changelog/plans/vX.Y.Z.json --apply
   go run ./scripts/releasenotes validate --version X.Y.Z --previous vPREVIOUS --dir changelog/vX.Y.Z
   ```

5. 创建 release-prep PR。PR 自身的 `release-note` 声明和 PR URL 也必须进入该版本 plan；如 PR 编号只能在创建后确定，先创建 PR，再补齐 plan/双语 notes 并等待 required CI/review。tag 只能指向已合并且已通过最终审计的 release-prep commit。

## 历史同步与发布

对每个历史版本先 dry-run，再在授权后加 `--apply`。创建/推送 tag 前再次确认远端不存在同名 tag；已推送的
稳定 tag 视为不可变，失败时不得删除、覆盖或创建替代 tag，除非用户明确授权该破坏性修复并说明影响。
发布后读取 GitHub Release 正文、tag target、draft 状态和全部资产，确认它与本地双语渲染一致。网络、
GitHub API、格式或来源验证失败必须直接报告；不得伪造成功、猜测 PR 号或静默改写其他 Release。本机
联网调用显式从 `GH_TOKEN` 读取凭据；不要打印、提交或把令牌写入计划文件。
