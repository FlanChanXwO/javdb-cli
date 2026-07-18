# Go SDK

[English](sdk.md) | [简体中文](sdk.zh-CN.md)

公开包：`github.com/FlanChanXwO/javdb-cli/javdb`

SDK 是内部签名 App API 客户端的薄封装。它**不会**自动读 `auth.json`——请自行传入 token 或调用 `Login`（凭证文件由 CLI 管理）。

## 安装

```bash
go get github.com/FlanChanXwO/javdb-cli/javdb@latest
```

## 创建客户端

```go
c, err := javdb.New(
    javdb.WithHost(javdb.HostMirror), // 或 HostMain / 绝对 URL
    javdb.WithProxy("http://127.0.0.1:7890"),
    javdb.WithToken(existingJWT),
    javdb.WithTimeout(20*time.Second),
    javdb.WithLang("en"),
    javdb.WithDeviceUUID(stableUUID),
)
```

## 登录

```go
ctx := context.Background()
token, err := c.Login(ctx, username, password)
uid, name, err := c.ResolveUserID(ctx)
c.SetToken(token)
```

`ResolveUserID` 需要有效登录（或已有 token）。若无法解析数字 user id 会直接失败。

## 常用方法

| 方法 | 用途 |
|------|------|
| `Search(ctx, q, SearchOptions)` | 关键词搜索 |
| `MovieDetail(ctx, id)` | 内部 id 取影片 |
| `ResolveMovieID(ctx, number)` | 番号 → 内部 id |
| `MovieMagnets(ctx, id)` | 磁力（需登录） |
| `FilterMagnets` / `PickBestMagnet` / `MagnetURI` | 客户端磁力筛选 |
| `Browse(ctx, BrowseOptions)` | 分类浏览 |
| `ResolveTags` / `LoadOrRefreshTaxonomy` | 标签别名 |
| `EntityMovies` / `ResolveEntity` / `EntityDetail` | 演员/系列/…/合集片单 |
| `WatchedMovies` / `WantMovies` / `Mark` / `Unmark` | 看過 / 想看 |
| `Collected` / `RecentViewed` | 收藏与最近浏览 |
| `RankingsMovies` / `RankingsActors` / `RankingsPlayback` | 排行 |
| `Top250` | TOP250（需登录） |
| `MyLists` / `ListInfo` / `RelatedLists` | 合集 |

列表结果多为 `SearchResult`（`map[string]json.RawMessage`），并提供：

```go
res, err := c.Search(ctx, "SSIS", javdb.SearchOptions{Zone: "censored", Limit: 10})
movies := res.Movies()
// 其它维度：res.Named("actors")
```

## 错误

```go
var ar *javdb.AuthRequired
if errors.As(err, &ar) {
    // token 缺失/过期 — 重新登录
}
var api *javdb.APIError
if errors.As(err, &api) {
    // 服务端 success:0
}
```

## 说明

- 默认主机为便于直连的公开镜像。
- 并发：每个 goroutine 使用独立 client 更安全（底层 HTTP 客户端共享）。
- CLI 二进制版本与 SDK 模块版本无关——用 Go modules 钉版本。
