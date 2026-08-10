# JavDB 版本化发布说明体系设计

## 目标

将 `javdb-cli` 的发布说明从根目录单文件迁移到可审计、双语、按版本存放的
`changelog/`，并让 feature PR、Release workflow 与 GitHub Release 共享同一来源。

本次同时改善项目发现性：GitHub description 与 README 明确说明项目提供 CLI、公开
Go SDK 与 Agent skill。项目当前没有 MCP server；任何文档或元数据都不得宣称支持 MCP。

## 非目标

- 不新增 MCP、下载器、Rust 组件或其他仅因与 `pixiv-cli` 对称而存在的层。
- 不改变 CLI、公开 SDK、认证、配置、输出或发布资产的现有行为。
- 不迁移 Pixiv 专属的 Rust、OAuth、ClawHub 与 Homebrew prepublish 工作流。

## 内容模型与目录

`changelog/` 是 release notes 的权威来源：

```text
changelog/
├── README.md                 # 英文索引、格式与发布流程
├── README.zh-CN.md           # 中文索引与流程
├── unreleased/
│   ├── en.md
│   └── zh-CN.md
├── plans/                     # 已审核、版本控制的 release-prep manifests
├── v0.1.0/{en.md,zh-CN.md}
├── v0.1.1/{en.md,zh-CN.md}
└── v0.2.0/{en.md,zh-CN.md}
```

每个版本有相同的英文与简体中文文件，按 `Added`、`Changed`、`Fixed`、`Security`、
`Documentation`、`Maintenance` 分组；空分组不写入。每个条目带对应 PR 或历史直接提交
链接。根目录 `CHANGELOG.md` 与 `CHANGELOG.zh-CN.md` 保留为旧链接兼容入口，只链接到
`changelog/`，不再保存版本正文。

每个 release-prep PR 在 `changelog/plans/vX.Y.Z.json` 提交一个受版本控制的 manifest。manifest
顶层必须包含与文件名相同的 `version`、前一版本 `previous_tag` 与版本级 `compare_url`；后者只
允许同仓库的 `compare/<previous_tag>...<version>` URL。每个 entry 具有唯一 `source`，并包含
`category`、`breaking`、`english`、`zh_cn` 与来源类型：合并 PR 或明确标注的历史直接提交。
`prepare` 与 `validate` 只接受这个 schema；英文和中文条目必须都来自该 manifest，人工翻译与
分组合并在 release-prep PR 中审核，而非从 feature PR 的英文摘要自动推断。

`v0.1.0` 是唯一的 bootstrap 例外：其 manifest 仍须包含 `version`，但 `previous_tag` 与
`compare_url` 必须为 `null`，所有 entries 均为同仓库的 `direct_commit`。validator 与 audit 对
此模式使用仓库起点至 `v0.1.0` 的明确范围；`v0.1.1` 起所有版本强制使用前一 tag 与 compare URL。

## Feature PR 契约

`.github/PULL_REQUEST_TEMPLATE.md` 增加唯一的 HTML release-note 声明。声明包含：

- `category`：标准分组或 `None`；
- `breaking`：`true` 或 `false`；
- `summary`：一条面向用户的英文摘要；
- `none_reason`：只在 `None` 时说明理由。

功能 PR 只携带声明，不直接修改 `changelog/unreleased/`。文档、测试、维护或无用户可见
影响的变更可使用 `None`，但必须有理由。release-prep manifest 的 PR source 必须与该声明
的 `category`、`breaking` 相符；`summary` 可被人工润色或翻译，但不能改变用户可见语义。PR
模板的其他检查项同步反映 SDK、文档、发行与凭据边界。

## 工具与执行流程

新增 `scripts/releasenotes` Go 命令，职责分为：

1. `audit --github`：只在显式联网时读取 GitHub PR 元数据和候选范围，报告缺失或无效声明、
   直接提交例外、首次贡献者与建议的 SemVer bump。
2. `prepare`：读取已审核的 `changelog/plans/vX.Y.Z.json`，预览或写入
   `changelog/vX.Y.Z/`，并清空 `unreleased/`。
3. `validate --offline`：不联网校验 manifest schema、版本目录、双语分类一致性、比较链接与
   允许的来源 URL 形式；pre-commit 只能调用此模式。
4. `sync-history`：对显式命名的历史版本预览或回写 GitHub Release 正文；正文按英文在前、中文
   在后渲染，使用本仓库内的版本化 notes。

entry source 仅允许 `https://github.com/FlanChanXwO/javdb-cli/pull/<number>` 或同仓库的 commit URL；
tag compare URL 只能位于 manifest 顶层，不能作为 entry source。`audit --github` 以 manifest 的
`previous_tag..version` 作为唯一候选范围，验证 PR 已合并且 merge commit 位于该范围内；历史直接
提交必须在 manifest 中标为 `direct_commit`，并验证 commit 位于对应 tag 范围。禁止以任意外部
URL 作为 source。

工具必须区分只读预览与显式 `--apply` 写入。任何网络、GitHub API、格式或来源校验失败均返回
真实错误，绝不发布空正文或猜测 PR 编号。

## CI、发布与本地门禁

- CI 保留始终触发、名称固定的 `quality` 与 `platform smoke gate` jobs。先由
  `classify_changes` 输出 `docs_only`；仅当改动全部落在 `README*.md`、`CONTRIBUTING*.md`、
  `CHANGELOG*.md`、`docs/**`、`changelog/**`、`skills/**`、`.agents/skills/**`、
  `.github/ISSUE_TEMPLATE/**` 或 `.github/PULL_REQUEST_TEMPLATE.md` 时，才可判为文档专属。
  文档专属改动仅运行文档契约与 release-note declaration 检查，平台矩阵可跳过，但
  `platform smoke gate` 必须显式把该跳过判为成功。任何不在此白名单内的改动运行完整门禁。
  workflow policy 测试按该集合断言分类器、固定 job 与分流关系。
- 需要 `audit --github` 的 PR/release-prep workflow 与 Release workflow 必须显式声明最小
  `pull-requests: read` 权限；Release 原有写入 job 另保留实际需要的 `contents: write`。workflow
  policy 测试断言这些权限，不能依赖匿名或 fork 环境的偶然读取。
- release-prep PR 以 manifest 的 `version`、`previous_tag` 和 `compare_url` 为唯一候选输入，并运行
  `prepare` dry-run、`validate --offline`、`audit --github` 与 release-body 渲染快照。CI 校验文件名、
  版本字段、比较链接和 PR 声明一致；只有该 PR 合并后的 commit 可创建同名 release tag。
- `release.yml` 在打包前重复校验 tag 对应的 `changelog/vX.Y.Z/`、manifest 与 GitHub 审计，再以
  同一渲染结果创建 GitHub Release；它只接受默认分支历史中的不可变 tag。
- `.pre-commit-config.yaml` 增加 release-note/文档结构本地检查。
- 维护者开发文档说明 feature PR、release-prep PR、tag、历史回写与显式授权边界。

## Repo-local Skills

新增 `.agents/skills/` 下的四项 Skills：

- `javdb-cli-review`：按维护者 review checklist 进行 finding-first 审查。
- `javdb-cli-docs`：按现有 locale 与文档路由维护文档。
- `javdb-cli-commit-message`：仅根据 staged changes 输出 Conventional Commit 标题。
- `javdb-cli-release-notes`：执行 PR metadata、release-prep、历史同步与 tag 发布的授权流程。

不创建 MCP 专用 Skill。各 Skill 只引用现有或本次新增的真实文件与命令。

## 历史迁移与外部更新

将根目录变更日志中 `v0.1.0`、`v0.1.1`、`v0.2.0` 的既有内容逐项迁入相应版本目录。历史条目
使用真实 tag 比较链接或标明 `direct_commit` 的 Git commit URL，不编造 PR 号。所有本地校验通过
后，`sync-history --apply --versions v0.1.0,v0.1.1,v0.2.0` 只允许更新当前仓库的这三个 GitHub
Release 正文；它以生成的临时 notes 文件提交，随后读取并逐字比对。不得创建新 tag、修改资产或
触碰其他 Release。

GitHub description 更新为 `Unofficial JavDB App API CLI, public Go SDK, and agent-ready automation skill.`；
执行 `gh repo edit FlanChanXwO/javdb-cli --description ...` 后立即读取 metadata 验证。此操作与历史
Release 回写同属外部写入，必须在用户授权后的最终阶段执行。

## 验收与测试

- 给 `scripts/releasenotes` 补充单元测试，覆盖缺失/重复声明、分类、版本、来源、双语渲染、
  离线/联网模式、预览与 `--apply` 的授权边界。
- 扩展文档/工作流测试，验证 PR 模板、根目录 changelog compatibility stubs、目录结构、CI 的
  docs-only 条件以及 Release workflow 调用的命令。
- 运行 `pre-commit run --all-files`、`go test ./...`、`sh scripts/build.sh` 及现有 workflow
  policy/文档测试。GitHub 历史回写后读取三个 Release 正文，确认与本地渲染逐字一致。

## 风险与控制

最大风险是历史发布正文、release tag 与本地 notes 不一致。为此所有外部写入均先 dry-run，再以
固定版本清单单独 `--apply`；每次调用报告目标版本和结果。Release workflow 只接受不可变 tag，
并在发布前验证该 tag 位于默认分支历史中；release-prep PR 将格式或来源错误前移至 tag 创建前。
