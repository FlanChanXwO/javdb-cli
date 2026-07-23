<div align="center">

# javdb-cli

**JavDB CLI · Go SDK · Agent skill**

[English](README.md) · [简体中文](README.zh-CN.md)

<p><a href="https://github.com/FlanChanXwO/javdb-cli/actions/workflows/ci.yml"><img alt="Quality gate" src="https://github.com/FlanChanXwO/javdb-cli/actions/workflows/ci.yml/badge.svg"></a> <a href="https://github.com/FlanChanXwO/javdb-cli/actions/workflows/platform-smoke.yml"><img alt="Platform smoke" src="https://github.com/FlanChanXwO/javdb-cli/actions/workflows/platform-smoke.yml/badge.svg"></a> <a href="https://github.com/FlanChanXwO/javdb-cli/releases/latest"><img alt="Release" src="https://img.shields.io/github/v/release/FlanChanXwO/javdb-cli?style=flat-square"></a> <a href="go.mod"><img alt="Go" src="https://img.shields.io/github/go-mod/go-version/FlanChanXwO/javdb-cli?style=flat-square"></a> <a href="LICENSE"><img alt="License" src="https://img.shields.io/github/license/FlanChanXwO/javdb-cli?style=flat-square"></a></p>

[安装](#安装) · [快速开始](#60-秒快速开始) · [使用入口](#选择使用入口) · [文档](#文档) · [参与贡献](CONTRIBUTING.zh-CN.md)

</div>

`javdb-cli` 是独立开发的非官方命令行客户端与 Go SDK，用于访问
[JavDB](https://javdb.com) App JSON API。它为用户、Coding Agent 和 Go 应用提供统一的
搜索、目录导航、影片详情、磁力、排行与已认证用户列表能力；它不是网页爬虫、不包含 MCP，
也不与 JavDB 或其运营方存在隶属、认可或其他关联。请仅在遵守 JavDB 条款和所在地适用法律的
前提下使用。

## 为什么选择 javdb-cli？

- **一致的能力面**——CLI 与公开 Go SDK 都覆盖搜索、详情、标签、浏览、实体片单、磁力、
  排行、TOP250、合集和已认证的想看/看过数据。
- **API 客户端而非爬虫**——命令通过 App JSON API 请求，并显式选择主机与代理；失败会原样
  显示，不会伪装成空结果。
- **适合 Agent 导航**——`detail` 提供稳定图 ID，命令支持 JSON 输出，并随仓库提供
  [操作 skill](skills/javdb-cli/SKILL.md)，可安全完成多步自动化。
- **本地多账号凭据**——用户名、密码和 token 保存在 `~/.javdb-cli/auth.json`（支持 POSIX
  权限的平台使用 `0600`）；常规命令输出绝不会打印 JWT。
- **有意识的状态变更**——看过/想看标记、默认账号变更、配置写入和标签缓存刷新都是显式操作。
- **原生发布证据**——CI 会在 macOS、Linux、Windows 的 amd64/arm64 平台构建、打包、解包并
  冒烟运行二进制。

## 安装

### Homebrew（macOS 与 Linux 推荐）

```bash
brew install FlanChanXwO/tap/javdb-cli
```

后续更新：

```bash
brew update
brew upgrade javdb-cli
```

启用 tap 部署后，Formula 会在验证完成的发版后更新。

### Go

请使用精确的已发布 tag：

```bash
go install github.com/FlanChanXwO/javdb-cli/cmd/javdb@v0.2.0
```

### Release 压缩包或源码构建

从 [GitHub Releases](https://github.com/FlanChanXwO/javdb-cli/releases) 下载与平台对应的
压缩包，先用同附的 `checksums.txt` 校验；再将 `javdb`（Windows 为 `javdb.exe`）放入 PATH 中的
目录。

构建当前 checkout 时，先安装 `go.mod` 指定的 Go 版本，再运行：

```bash
sh scripts/build.sh
./build/javdb version --json
```

发布契约覆盖 macOS、Linux、Windows 的 amd64 与 arm64。可复现目标构建及归档内容见
[开发指南](docs/maintainers/development.md#构建打包与平台)。

### 更新

已发布的安装可先检查，再安装新版：

```bash
javdb update --check
javdb update
```

`update` 会识别 `javdb` 由 Homebrew、`go install` 还是 Release 压缩包管理，并选择对应渠道。
压缩包更新只下载当前 OS/架构的资产，用同一 Release 的 `checksums.txt` 校验 SHA-256，再验证下载
二进制报告的版本，最后替换可执行文件。`--check --json` 是机器可读且不写入的检查形式；
`--prerelease` 会纳入预发布 tag。Homebrew 安装只支持 stable release。现有的 `--proxy URL` 同样
适用于 Release 检查。

### 让 Coding Agent 安装

把以下 prompt 复制给能访问本机终端的 Codex、Claude Code、Cursor 或其他 Coding Agent：

```text
请为这台机器安装 https://github.com/FlanChanXwO/javdb-cli 的最新 stable 版本。检测操作系统和架构，只下载官方 GitHub Release 资产，必须先用 checksums.txt 中对应的 SHA-256 校验通过才安装；创建或修改任何 PATH 目录前先询问；绝不读取或输出 ~/.javdb-cli/auth.json 或凭据；最后运行 javdb version --json 验证，并报告安装版本和所有变更文件。

同时从相同 stable release tag 安装完整的 skills/javdb-cli/ 目录到我确认的 Agent skills 目录。不要猜测 skills 路径，不要使用 main 分支的 skill 内容，并保留全部 references 文件。
```

## 60 秒快速开始

```bash
# 交互式登录；CLI 不会打印 JWT。
javdb auth login
javdb auth check --json

# 搜索影片，再读取详情中的图 ID 以便后续导航。
javdb search SSIS-589 --limit 5 --json
javdb detail SSIS-589 --json

# 按标签浏览，并获取经过筛选的磁力列表（需要认证）。
javdb browse --tag 巨乳 --main m --limit 20 --json
javdb magnets SSIS-589 --cnsub --hd --json
```

运行 `javdb --help`，或阅读[完整命令参考](docs/zh-CN/cli-reference.md)，查看所有命令、flag、
配置键与认证要求。

## 选择使用入口

### CLI

交互时使用表格输出；命令支持时，Agent 或脚本可使用 `--json` 获取稳定字段：

```bash
javdb rankings movies --period week
javdb actor 山手梨愛 --main m --has-magnets --json
javdb lists search 巨乳 --zone all --json
```

全局 `--proxy URL` 与 `--host mirror|main` 仅影响本次命令。依赖持久化设置前先执行
`javdb config get` 查看有效配置。

### Go SDK

```go
c, err := javdb.New(javdb.WithHost(javdb.HostMirror))
if err != nil {
	panic(err)
}

res, err := c.Search(context.Background(), "SSIS-589", javdb.SearchOptions{Limit: 5})
if err != nil {
	panic(err)
}
fmt.Println(len(res.Movies()))
```

导入 `github.com/FlanChanXwO/javdb-cli/javdb`。[SDK 指南](docs/zh-CN/sdk.md)
说明公开模型、client options 与调用方职责。

### Agent skill

仓库提供 [skills/javdb-cli](skills/javdb-cli/SKILL.md)，为 Coding Agent 定义专用操作 skill。
它规定了凭据处理、确认边界、命令级参数、JSON/错误处理与搜索到详情的导航。仅在明确的 JavDB
任务中加载，并在执行前用 `javdb <command> --help` 核对参数。

## 认证与凭据安全

推荐用 `javdb auth login` 配置账号。用户名、密码与 session token 会保存于本地多账号存储
`~/.javdb-cli/auth.json`（支持 POSIX 权限的平台使用 `0600`）。不得提交、打印、粘贴或上传
该文件、密码或 JWT。

```bash
javdb auth list
javdb auth use USER_ID
javdb auth check --json
```

`javdb auth check` 会验证默认 token 而不会打印它。默认会明确报告 token 过期。`auto_relogin`
是 opt-in，且会用已保存密码重登一次；仅在接受这一行为后开启：

```bash
javdb config set auto_relogin true
```

磁力、TOP250 和用户列表需要默认已认证账号。`mark`/`unmark`、账号变更以及
`config set`/`unset` 会修改服务端或本地状态，应审慎使用。

## 文档

| 文档 | 用途 |
| --- | --- |
| [命令参考](docs/zh-CN/cli-reference.md) | 命令、flag、认证、配置与常见流程 |
| [Agent 操作 skill](skills/javdb-cli/SKILL.md) | Agent 路由、密钥、写操作、检索与错误 |
| [Go SDK](docs/zh-CN/sdk.md) | 公开 client、模型和 options |
| [架构说明](docs/maintainers/architecture.md) | 包边界与运行流程 |
| [开发指南](docs/maintainers/development.md) | 工具链、测试、平台构建、打包与发版 |
| [文档导航](docs/index.md) | 多语言公开契约与维护者指南 |
| [贡献指南](CONTRIBUTING.zh-CN.md) | 本地质量门与贡献规范 |
| [更新日志](CHANGELOG.zh-CN.md) | 用户可感知变更 |

## 参与贡献

欢迎提交 bug、文档修复、测试和聚焦功能。发起 pull request 前请阅读
[CONTRIBUTING.zh-CN.md](CONTRIBUTING.zh-CN.md)；较大或影响兼容性的变更请先讨论。

## 许可证

[MIT](LICENSE) © FlanChanXwO
