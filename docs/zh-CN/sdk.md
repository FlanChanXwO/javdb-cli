# javdb Go SDK

[文档导航](../index.md) · [English](../en/sdk.md)

公开包为 `github.com/FlanChanXwO/javdb-cli/javdb`。它与 CLI 共享同一远程能力面，
不会为应用自动读取 `auth.json` 或管理本机账号。

## 安装与创建

应用应钉住已发布的精确 tag：

```bash
go get github.com/FlanChanXwO/javdb-cli/javdb@vX.Y.Z
```

```go
client, err := javdb.New(
    javdb.WithHost(javdb.HostMirror),
    javdb.WithProxy("http://127.0.0.1:7890"),
    javdb.WithToken(existingJWT),
    javdb.WithLang("en"),
    javdb.WithDeviceUUID(stableDeviceUUID),
)
if err != nil {
    return err
}
```

只有调用方明确需要时才选择 `HostMain` 或绝对 URL。`WithTimeout` 配置 HTTP client。
若需要跨进程稳定 device identity，可使用 `javdb.LoadOrCreateDeviceUUID(path)`，再把返回值传给
`WithDeviceUUID`。

## 登录

```go
ctx := context.Background()
token, err := client.Login(ctx, username, password)
if err != nil {
    return err
}
client.SetToken(token)
userID, username, err := client.ResolveUserID(ctx)
```

凭据与持久化由调用方负责。JWT 或密码不得进入日志、panic、错误包装或测试 fixture。

## 能力

| 能力 | 方法 |
| --- | --- |
| 发现 | `Search`、`MovieDetail`、`ResolveMovieID`、`Browse`、`ResolveTags` |
| 实体图 | `ResolveEntity`、`EntityDetail`、`EntityMovies`、`AllEntityMovies` |
| 磁力 | `MovieMagnets`、`FilterMagnets`、`PickBestMagnet`、`MagnetURI` |
| 排行 | `RankingsMovies`、`RankingsActors`、`RankingsPlayback`、`Top250` |
| 个人状态 | `WatchedMovies`、`WantMovies`、`Mark`、`Unmark`、`Collected`、`RecentViewed` |
| 合集 | `MyLists`、`ListInfo`、`RelatedLists` |
| 标签目录 | `RefreshTagTaxonomy`、`LoadOrRefreshTaxonomy` |

许多列表操作返回 `SearchResult`，可按响应维度取值：

```go
result, err := client.Search(ctx, "SSIS", javdb.SearchOptions{
    Zone:  "censored",
    Limit: 10,
})
movies := result.Movies()
actors := result.Named("actors")
```

更新看过/想看状态及刷新本机公开标签缓存都是 mutation；只有在应用获得明确授权时才调用。

## 错误与兼容性

```go
var authRequired *javdb.AuthRequired
if errors.As(err, &authRequired) {
    // 通过调用方选择的凭据流程重新认证。
}

var apiError *javdb.APIError
if errors.As(err, &apiError) {
    // App API 返回 success:0 的服务端失败。
}
```

公开包才是支持的集成边界。`internal/` 路径、wire payload、签名细节和 `Client.API` escape hatch
都不是稳定外部契约；集成方应钉住模块版本并只使用已记录的方法与类型。
