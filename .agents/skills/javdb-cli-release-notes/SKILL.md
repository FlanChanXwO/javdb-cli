---
name: javdb-cli-release-notes
description: Manage javdb-cli bilingual changelog releases, historical GitHub Release synchronization, and approved version releases.
---

# javdb-cli release notes and release workflow

在 release-prep、changelog、tag、GitHub Release 或历史 Release 正文同步时使用。
先读取 `docs/maintainers/development.md`；它是长期 policy reference。

## 授权边界

SemVer 建议不是发布授权。创建或合并 release-prep PR、创建或推送 tag、触发发布、更新 GitHub
description 或执行 `sync-history --apply` 前，必须在当前会话取得明确授权，说明目标版本、范围与
预期影响。历史同步必须列出允许更新的版本，绝不创建 tag 或替换资产。

## Release-prep

1. 确定目标版本和前一稳定 tag：

   ```bash
   previous=$(sh scripts/previous-release-tag.sh vX.Y.Z)
   ```

2. 如需核对候选范围，可生成只读审计报告：

   ```bash
   go run ./scripts/releasenotes audit \
     --repo FlanChanXwO/javdb-cli \
     --from "$previous" \
     --to COMMIT_OR_TAG \
     --output /tmp/javdb-release-audit.json
   ```

3. 在 release-prep PR 中直接编辑 `changelog/vX.Y.Z/en.md`、`changelog/vX.Y.Z/zh-CN.md`，同步
   `changelog/README.md` 和 `changelog/README.zh-CN.md`。每个条目必须带 PR 或 direct-commit 来源，
   两个 locale 的来源集合必须一致。`changelog/unreleased/` 只是可选人工草稿区，不会被发布 workflow 读取。

4. 运行本地门禁：

   ```bash
   sh scripts/test-releasenotes.sh
   go run ./scripts/releasenotes validate --version X.Y.Z --previous "$previous" --dir changelog/vX.Y.Z
   ```

5. 创建 release-prep PR，运行 required CI 并等待审查；tag 只能指向已合并的 release-prep commit。

`audit` 只确认 changelog 中引用的来源属于发布区间，不要求区间内每个 PR/commit 都必须出现在 notes 中；
PR 正文不再承载或声明发布分类、摘要或版本升级信息。

## 历史同步与发布

对每个历史版本先 dry-run，再在授权后加 `--apply`。发布后读取 GitHub Release 正文，确认它与本地
双语渲染一致。可用命令为 `validate`、`audit`、`render` 和 `sync-history`。网络、GitHub API、格式或
来源验证失败必须直接报告；不得伪造成功、猜测 PR 号或静默改写其他 Release。本机联网调用显式从
`GH_TOKEN` 读取凭据；不要打印、提交或把令牌写入计划文件。
