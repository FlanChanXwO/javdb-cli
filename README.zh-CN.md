# javdb

[English](README.md) | [简体中文](README.zh-CN.md)

面向 [JavDB](https://javdb.com) **App JSON API** 的非官方命令行客户端（Go 实现）。

| | |
|---|---|
| **二进制** | `javdb` |
| **模块** | `github.com/FlanChanXwO/javdb-cli` |
| **许可证** | [MIT](./LICENSE) |
| **Homebrew** | `brew install FlanChanXwO/tap/javdb-cli`（发版后） |

这是 App API 的**客户端**，不是网页爬虫，也不包含 MCP。

## 免责声明

**javdb 是非官方第三方工具，与 JavDB 及其运营方无任何关联、授权或背书关系。**

- 你须自行遵守 JavDB 服务条款以及所在地法律法规。
- 账号密码与会话令牌会**保存在本机** `~/.javdb-cli/` 目录，风险自负。建议使用专用账号，切勿把密钥提交进仓库。
- 本软件按「现状」提供，**不作任何担保**。作者不对封号、数据丢失、法律纠纷或其他损害负责。
- 仅供个人学习 / 研究用途。请勿用于骚扰他人或滥用服务。

## 安装

### 源码编译

```bash
git clone https://github.com/FlanChanXwO/javdb-cli.git
cd javdb-cli
sh scripts/build.sh
./build/javdb version
```

需要 Go **1.26+**（见 `go.mod`）。

### go install

```bash
go install github.com/FlanChanXwO/javdb-cli/cmd/javdb@latest
# 发版后可钉死标签：
# go install github.com/FlanChanXwO/javdb-cli/cmd/javdb@v0.1.0
```

### Homebrew

```bash
brew install FlanChanXwO/tap/javdb-cli
```

## 快速开始

```bash
# 登录（可省略 -u/-p 进入交互；不会打印 JWT）
javdb auth login -u USER -p PASS
javdb auth list
javdb auth check --json

# 搜索 / 详情 / 磁力
javdb search SSIS-589 --limit 5
javdb detail SSIS-589
javdb magnets SSIS-589 --best --json

# 分类浏览与实体片单
javdb tags --zone censored
javdb browse --tag 巨乳 --main m --limit 10
javdb actor 山手梨愛 --main m --has-magnets
javdb list RZ8Bm --limit 5

# 用户列表（需登录）
javdb watched
javdb want
javdb recent
javdb mark SSIS-589 --want

# 排行 / TOP250 / 合集
javdb rankings movies --period week
javdb top250 --limit 20
javdb lists
javdb lists search 巨乳
```

全局参数：`--proxy URL`、`--host mirror|main`（默认 **mirror**）。

配置与凭证：

| 路径 | 用途 |
|------|------|
| `~/.javdb-cli/auth.json` | 多账号 用户名/密码/token（权限 `0600`） |
| `~/.javdb-cli/config.toml` | host、proxy、`auto_relogin`、lang |
| `~/.javdb-cli/tags-*.json` | 各分区标签分类缓存 |

```bash
javdb config set auto_relogin true   # JWT 过期时可选静默重登
javdb config set host mirror
```

## 公开 Go SDK

```go
package main

import (
	"context"
	"fmt"

	"github.com/FlanChanXwO/javdb-cli/javdb"
)

func main() {
	c, err := javdb.New(javdb.WithHost(javdb.HostMirror))
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	if _, err := c.Login(ctx, "user", "pass"); err != nil {
		panic(err)
	}
	res, err := c.Search(ctx, "SSIS-589", javdb.SearchOptions{Limit: 5})
	if err != nil {
		panic(err)
	}
	fmt.Println(len(res.Movies()))
}
```

更多见 [docs/sdk.zh-CN.md](./docs/sdk.zh-CN.md)。

## 文档

| 文档 | 说明 |
|------|------|
| [docs/index.zh-CN.md](./docs/index.zh-CN.md) | 文档导航 |
| [docs/usage.zh-CN.md](./docs/usage.zh-CN.md) | 命令参考 |
| [docs/development.zh-CN.md](./docs/development.zh-CN.md) | 构建、测试、目录结构 |
| [docs/sdk.zh-CN.md](./docs/sdk.zh-CN.md) | 公开包说明 |
| [CONTRIBUTING.zh-CN.md](./CONTRIBUTING.zh-CN.md) | 贡献指南 |
| [CHANGELOG.zh-CN.md](./CHANGELOG.zh-CN.md) | 更新日志 |

英文版为同名不带 `.zh-CN` 的文件。

## 测试

```bash
go test ./...
go test -race ./...
go vet ./...
sh scripts/build.sh
```

## 许可证

[MIT](./LICENSE) © FlanChanXwO
