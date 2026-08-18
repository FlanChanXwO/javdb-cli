# 为 javdb-cli 贡献

[English](CONTRIBUTING.md) | 简体中文

感谢你帮助改进 `javdb-cli`。我们欢迎聚焦的 bug report、文档修复、测试和边界清晰的功能。

## 开始之前

- 先检索已有 issue 和 pull request，避免重复提交。
- 大功能、public API 变更、新依赖或认证变更应在实施前讨论。
- 不要在 issue、fixture、commit 或 CI 日志中包含密码、JWT、`~/.javdb-cli/auth.json` 内容、tag cache、机器配置或私有 API 响应。
- 保持改动聚焦；无关清理更适合单独提交 pull request。

## 开发环境

受支持的源码构建使用 Go `1.26.3` 与标准 Go toolchain，没有 C 或 native 依赖。

在仓库根目录构建和测试：

```bash
go test ./...
sh scripts/build.sh
./build/javdb --version
```

opt-in 真实 API 测试、发布门禁和平台细节见[开发流程](docs/maintainers/development.md)。

## 架构边界

- `cmd/javdb` 只委托 `internal/cli`；保持二进制精简，把 Cobra、输入与输出放在 `internal/cli`。
- 远程 JavDB 操作只通过顶层 public `sdk/` SDK（`package javdb`）暴露；协议实现在 `internal/javdb/appapi` 与 `internal/javdb/protocol/*`。
- `internal/config` 管理本机配置；`internal/storage/auth` 与 `internal/storage/tags` 管理本机状态；`internal/update` 负责显式更新的 Release 检查、来源识别与校验替换。
- 文件应聚焦于一个职责或少数紧密相关职责。

修改这些边界前，请阅读[架构说明](docs/maintainers/architecture.md)与仓库 [AGENTS.md](AGENTS.md)。

## 使用测试驱动开发

代码变更采用 red-green-refactor：

1. 添加一个会因目标行为尚未实现而失败的聚焦测试。
2. 实现让它通过的最小完整变更。
3. 在不改变已验证公开行为的前提下重构。
4. 先运行聚焦测试，再运行相关回归。

可行时通过 public boundary 测试公开行为。不得把真实的认证、网络、JavDB API、文件系统或编码失败隐藏为空成功或静默 fallback；不得增加无依据的 timeout、截断、分页上限、重试限制或隐藏降级。

真实 JavDB App API canary 均为 opt-in。未经用户明确授权，不得使用其本地账号运行；也不要把真实 token 放入可能写入 shell history 的命令行。

## 文档

修改命令、flag、SDK API、配置键、环境变量、输出契约、认证流程、代理行为或已知限制时，在同一 pull request 同步文档。

- 保持 `README.md` 与 `README.zh-CN.md` 的行为语义对应。
- 保持 `docs/en/` 与 `docs/zh-CN/` 下两个语言版本的行为语义对应；不得用未翻译占位内容冒充对应语言。
- 按文件职责更新 `docs/sdk.md` / `docs/sdk.zh-CN.md`、`docs/maintainers/architecture.md` 或 `docs/maintainers/development.md`。
- 发布说明由 release-prep PR 直接维护在 `changelog/vX.Y.Z/{en.md,zh-CN.md}`；英文与简体中文条目必须对应，每条都要包含 PR 或 direct commit 来源，并同步更新两个 changelog 索引。具体流程见[开发流程](docs/maintainers/development.md)。
- CLI 命令、flag 或安全语义变化时检查 `skills/javdb-cli/`。

稳定规则只在一个权威文档中定义，其他位置应链接过去，避免复制大段内容。

## Pull request checklist

请求 review 前确认：

- [ ] 改动保持聚焦，并说明了用户可感知行为。
- [ ] 新增或修改代码有聚焦测试，并且测试曾先证明失败。
- [ ] `go test ./... -count=1` 通过。
- [ ] `go vet ./...` 通过。
- [ ] `sh scripts/build.sh` 通过。
- [ ] 发布敏感检查通过（`scripts/test-package-release.sh`、`test-homebrew-formula.sh`、`test-workflows.sh`）。
- [ ] pre-commit 可用时，`python -m pre_commit run --all-files` 通过。
- [ ] `git diff --check` 通过。
- [ ] 需要同步的英文与简体中文文档已对应。
- [ ] 未包含凭据、本地状态或机器相关产物。

Commit message 推荐使用 Conventional Commits，例如 `feat(cli): add account selection` 或 `docs: clarify anonymous fallback`。除非未来规范明确要求，项目不要求 CLA、DCO sign-off 或 signed commit。

## 许可证

提交贡献即表示你同意该贡献可按仓库的 [MIT License](LICENSE) 分发。
