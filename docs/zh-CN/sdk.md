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

## 显式自动选线

`SelectAutoHost` 显式探测 App API 并返回最快主机 URL；`javdb.New` 自身从不选线，
`javdb.New(javdb.WithHost("auto"))` 不会自动联网。选线后再用具体主机构造 client：

```go
result, err := javdb.SelectAutoHost(ctx, javdb.AutoHostOptions{
    PreferredHost: cachedHost, // 可选；上次已验证、可复用的主机
    Proxy:         proxy,
    DeviceUUID:    deviceUUID,
    Lang:          "en",
})
if err != nil {
    return err
}
client, err := javdb.New(javdb.WithHost(result.Host), javdb.WithToken(token))
if err != nil {
    return err
}
```

`AutoHostOptions.PreferredHost` 会被优先验证。验证成功时，SDK 立即原样返回该主机并令
`result.ReusedPreferred == true`，不会继续探测或排序其他候选，调用方也无需重写线路缓存。
只有 preferred 主机不可复用时，SDK 才发现并探测候选，返回最快的成功主机并令
`ReusedPreferred == false`。SDK 不负责持久化任何结果。`Latency` 是返回主机单次
`/startup` 请求耗时。选线探测使用零重试，延迟样本不会被重试污染。
`AutoHostOptions.Timeout` 约束每次探测请求（包括 preferred 主机验证）；零值沿用 transport
既有的 20 秒默认值。context 取消会立即中止选线。

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

## 以图搜番

SDK 通过同一个 `Client` 暴露图片反搜与严格 JavDB 联动：

```go
result, err := client.SearchByImage(ctx, javdb.ReverseSearchRequest{
    Image:    imageBytes, // 原始 JPEG/PNG/WEBP，≤ 8 MiB
    Filename: "frame.jpg",
    Source:   javdb.ReverseSearchSource{Name: "builtin"}, // 或声明式外部 HTTP source
}, javdb.ImageSearchOptions{})
if err != nil {
    // provider 顶层失败；绝不伪造空结果。
}
for _, match := range result.Matches {
    // match.Candidate.VideoCode、match.MovieID、match.Movie、match.Error
}
```

- `ReverseSearch` 把原始图片上传到所选 source（内置 AVScan 或声明式外部
  HTTP(S) source），返回规范化候选与帧；multipart 字段固定为 `file`。
- `SearchByImage` 对每个候选并发执行大小写不敏感、完整相等的严格番号匹配
  （`ResolveMovieIDExact`，不回退首项）并恢复 provider 顺序；单候选失败是
  `ImageSearchError`，绝不中止整批。
- `ReverseSearchCache` 是可注入接口（按原图 SHA-256 作为 key 的 `Get`/`Put`）；
  SDK 绝不读取 `~/.javdb-cli`。缓存命中跳过 provider，`BypassCache` 按请求禁用。
  缓存不得保存原图、鉴权 header 或 JavDB 详情。
- `ReverseSearchOptions` 配置重试（最多三次总请求）、30s/60s 退避与 60s 单请求
  超时；`WithReverseSearch` 注入，`javdb.New` 本身不联网。
- `WithProxy` 的代理同时用于 provider 请求。

隐私与网络边界：反搜会把你的图片上传到已配置的 provider（默认 AVScan），图片
URL 可能指向私网；服务端嵌入方必须自行施加出口边界。
