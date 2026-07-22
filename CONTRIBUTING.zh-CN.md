# 贡献指南

[English](CONTRIBUTING.md) | [简体中文](CONTRIBUTING.zh-CN.md)

感谢你参与改进 `javdb-cli`。较大改动前，请先阅读[架构说明](docs/maintainers/architecture.md)
与[开发指南](docs/maintainers/development.md)。

## 本地环境

```bash
git clone https://github.com/FlanChanXwO/javdb-cli.git
cd javdb-cli
go test ./...
sh scripts/build.sh
./build/javdb version --json
```

- 使用 `go.mod` 声明的 Go 版本。
- 不提交 `~/.javdb-cli/auth.json`、密码、JWT、tag cache 或机器相关配置。
- 优先离线测试。真机 API 抽查可能使用本机凭据，必须有明确授权，且不得输出 secret。

## 项目地图

```text
cmd/javdb/                         # 二进制入口
javdb/                             # 公开 SDK facade
internal/cli/                      # Cobra 输入/输出 adapter
internal/javdb/appapi/             # App JSON API adapter
internal/javdb/protocol/{httpx,signature}/
internal/config/                   # 配置路径与运行时合并
internal/storage/{auth,tags}/      # 本机状态
internal/buildinfo/                # linker 元数据
scripts/                           # 构建、打包和策略检查
skills/javdb-cli/                  # 产品操作 skill
docs/en/, docs/zh-CN/              # 多语言公开契约
docs/maintainers/                  # 架构、开发、ADR 与 agent 规则
```

CLI 的远程操作必须通过公开 `javdb` facade。不要把协议实现路径暴露为 SDK API，也不要仅为了
模仿 pixiv-cli 而创建不存在职责的空层。

## 改动要求

1. 除非明确记录兼容性变更，否则保持命令、flag、JSON 字段和文本输出稳定。
2. 行为改动补充聚焦测试；掩码、过滤、参数构造等纯逻辑优先表驱动测试。
3. 认证失败应清晰可见，且不得泄露凭据。
4. 行为变化时同步更新两个公开 locale、README、operator skill 与对应维护者文档。
5. 用户可感知的新增、修复、移除或安全变更写入两个 changelog 的 `Unreleased`。

## 发起 Pull Request 前

```bash
go test ./...
go test -race ./...
go vet ./...
sh scripts/build.sh
sh scripts/test-package-release.sh
sh scripts/test-homebrew-formula.sh
sh scripts/test-workflows.sh
sh scripts/test-documentation.sh
sh scripts/test-architecture.sh
pre-commit run --all-files
```

至少运行与改动相关的检查；涉及发布或大范围重构时运行完整列表。提交应聚焦改动本身；大型或
兼容性敏感的改动先讨论。

## 许可证

贡献代码即表示你同意以 [MIT License](LICENSE) 授权你的贡献。
