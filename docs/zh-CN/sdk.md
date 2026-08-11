# javdb Go SDK

[文档导航](../index.md) · [English](../en/sdk.md)

公开 SDK 的导入路径为 `github.com/FlanChanXwO/javdb-cli/sdk`，并声明为
`package javdb`。它与 CLI 共享同一远程能力面，不会为应用自动读取 `auth.json` 或管理本机账号。

## 安装与创建

应用应钉住已发布的精确 tag：

```bash
go get github.com/FlanChanXwO/javdb-cli/sdk@vX.Y.Z
```

## 从 `/javdb` 迁移

这是一次破坏性导入路径迁移。将每个 import declaration 中的
`github.com/FlanChanXwO/javdb-cli/javdb` 替换为
`github.com/FlanChanXwO/javdb-cli/sdk`，再解析包含本次迁移的发布版本依赖。
package 名、公开类型和已记录的方法仍为 `javdb`，因此 `javdb.New` 等选择器无需修改。
旧的 `/javdb` 导入路径不再受支持。

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
| 评论 | `MovieComments` |
| 本地媒体下载 | `DownloadMovieMedia`、`MovieMediaDownloadOptions`、`MovieMediaDownloadResult` |
| 实体图 | `ResolveEntity`、`EntityDetail`、`EntityMovies`、`AllEntityMovies` |
| 磁力 | `MovieMagnets`、`FilterMagnets`、`PickBestMagnet`、`MagnetURI` |
| 排行 | `RankingsMovies`、`RankingsActors`、`RankingsPlayback`、`Top250` |
| 个人状态 | `WatchedMovies`、`WantMovies`、`Mark`、`Unmark`、`Collected`、`RecentViewed` |
| 合集 | `MyLists`、`ListInfo`、`RelatedLists` |
| 标签目录 | `RefreshTagTaxonomy`、`LoadOrRefreshTaxonomy` |

`RankingsMovies` 与 `RankingsPlayback` 接受 `censored`、`uncensored`、`western`、
`fc2` 等分区名称，也接受已转换的数字字符串；已知名称会在请求前归一化。三个排行方法均接受
`day`、`week`、`month`，也接受 API 形式的 `daily`、`weekly`、`monthly`。
`RankingPeriod` 提供短周期到 API 周期的映射；`ActorPeriod` 作为废弃兼容别名继续保留。

许多列表操作返回 `SearchResult`，可按响应维度取值：

```go
result, err := client.Search(ctx, "SSIS", javdb.SearchOptions{
    Zone:  "censored",
    Limit: 10,
})
movies := result.Movies()
actors := result.Named("actors")
```

`MovieComments(ctx, movieID, page, limit)` 只请求一页，绝不会遍历后续页。非正值会使用第 `1` 页、
每页 `20` 条，与 CLI 的单页默认语义一致。

只有在调用方已明确选择新的本地路径时才使用 `DownloadMovieMedia`：

```go
downloaded, err := client.DownloadMovieMedia(ctx, movieID, javdb.MovieMediaDownloadOptions{
    PreviewImagePath: "/chosen/output/preview-0.jpg", // 只取 preview_images[0]
    PreviewVideoPath: "/chosen/output/preview.ts",
})
if err != nil {
    return err
}
fmt.Println(downloaded.PreviewImageBytes, downloaded.PreviewVideoBytes)
```

每个非空路径选择一个媒体项。`PreviewImagePath` 始终只取首张预览图，不会遍历后续图片；图片会在
写入前校验。视频路径支持已结束的单媒体 HLS playlist（含 AES-128）；master、byte-range、
fragmented MP4、未结束或直播 playlist 会返回错误。所有输出路径必须互异、父目录必须已存在，且
目标文件不得存在。

更新看过/想看状态及刷新本机公开标签缓存都是 mutation；只有在应用获得明确授权时才调用。
媒体下载会写入本地文件，也必须由应用用户明确指定目标路径。

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
