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
| 本地影片资源 | `MovieAssets`、`DownloadMovieAsset`、`MovieAsset`、`MovieAssetsFromDetail`、`MovieAssetDescriptions`、`ImageAssetFormat` |
| 实体图 | `ResolveEntity`、`EntityDetail`、`EntityMovies`、`AllEntityMovies` |
| 磁力 | `MovieMagnets`、`FilterMagnets`、`PickBestMagnet`、`RankMagnets`、`MagnetURI` |
| 排行 | `RankingsMovies`、`RankingsActors`、`RankingsPlayback`、`Top250` |
| 个人状态 | `WatchedMovies`、`WantMovies`、`Mark`、`Unmark`、`Collected`、`RecentViewed` |
| 合集 | `MyLists`、`ListInfo`、`RelatedLists` |
| 标签目录 | `RefreshTagTaxonomy`、`LoadOrRefreshTaxonomy` |

`ResolveMovieID(ctx, number)` 会去除首尾空白，并优先进行大小写不敏感的完整番号匹配。
没有完整匹配时，唯一且无歧义的格式等价番号（例如分隔符差异）也可以接受；对应多个不同影片
ID 或只有模糊候选时返回错误，绝不会回退到搜索结果的首项。

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

`MovieAssets(ctx, movieID)` 以固定的顺序返回影片媒体资产的最小
`MovieAsset{Type, URL}` 序列：thumbnail、cover（详情提供时）、
全部预览图（优先 `large_url`，回退 `thumb_url`）、预览视频。缺失项直接跳过，因此序列长度
随影片而变。`Type` 只有 `"image"` 与 `"video"`。该模型刻意不携带尺寸、时长、
id/index/role 等元数据。

`DownloadMovieAsset(ctx, asset, target)` 把单个资产下载到精确路径并返回写入字节数。图片会先校验
（CDN 混淆时 XOR 解包，再魔数校验）并原子发布，不做任何格式转换。视频由 target 后缀决定输出格式：
`.ts` 保留解密校验后的 MPEG-TS；`.mp4` 通过纯 Go remux 生成 Fast Start MP4（ftyp → moov → mdat），
并在 HLS segment 边界对回退/重置的 DTS/PTS 做连续时间轴归一化；无 ffmpeg、无转码。其余后缀返回
`unsupported video output format`。不支持的编码（HEVC、AC-3 等）
明确失败。context 取消会中断所有阶段且不留下输出文件。

MP4 每段要求恰好一个 H.264 PID、至多一个 AAC-LC PID，且每个 ADTS 帧只有一个 raw data block；
只重封装 channel_configuration 1（mono）和 2（stereo），多轨、不支持的 AAC profile、block 数或
channel configuration 会明确失败。MP4 忽略 timed ID3。
TS 发布前校验媒体结构和视频时间轴，保留 ADTS 原文，不套用 MP4 的
profile/block/channel configuration 限制。

`ProbeMovieAssets(ctx, assets, options)` 使用固定 worker pool 探测选定的图片与预览视频资产，保持输入
顺序，并按 `Type + URL` 去重。`MovieAssetProbeOptions.Concurrency == 0` 使用默认值 `4`；负数会被拒绝。
单个资产错误（包括媒体请求自身超时）是 best-effort；只有调用方的 context 取消或到期才会让整个调用失败。
图片元数据为 `Width`/`Height`；
预览视频 `Duration` 为 HLS 时长四舍五入后的整数秒。方法只流式读取 header/首段，不下载完整图片或视频。

`MovieAssetsFromDetail` / `MovieAssetDescriptions` 把已取得的详情 map 映射为同一资产序列与仅用于
TTY 渲染的描述文本；描述文本不得进入机器输出。`ImageAssetFormat(path)` 报告本地图片的检测格式。

```go
assets, err := client.MovieAssets(ctx, movieID)
if err != nil {
    return err
}
for _, asset := range assets {
    if asset.Type == "video" {
        _, err := client.DownloadMovieAsset(ctx, asset, "/chosen/output/preview.mp4")
        return err
    }
}
```

本 API 只写入 thumbnail/preview 资源，不下载完整影片或磁力目标。这是一次破坏性变更：path-per-type
的 `DownloadMovieAssets`、`MovieAssetDownloadOptions` 与 `MovieAssetDownloadResult` 已移除，
不保留别名。

更新看过/想看状态及刷新本机公开标签缓存都是 mutation；只有在应用获得明确授权时才调用。
本地资源写入会创建本地文件，也必须由应用用户明确指定目标路径。

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
  `ImageSearchError`，绝不中止整批。设置 `ImageSearchOptions{SkipMovieDetail: true}`
  可仅解析 movie ID 而不获取完整详情——适用于只需 ID 的场景（如磁力查询）。
- `ReverseSearchCache` 是可注入接口（`Get`/`Put` 的 key 为
  `"<source>:<原图 SHA-256>"`，不同 provider 永不共享条目）；SDK 绝不读取
  `~/.javdb-cli`。缓存命中跳过 provider，`BypassCache` 按请求禁用。缓存不得
  保存原图、鉴权 header 或 JavDB 详情。
- `ReverseSearchOptions` 配置重试（最多三次总请求）、30s/60s 退避与 60s 单请求
  超时；`WithReverseSearch` 注入，`javdb.New` 本身不联网。
- `WithProxy` 的代理同时用于 provider 请求。

隐私与网络边界：反搜会把你的图片上传到已配置的 provider（默认 AVScan），图片
URL 可能指向私网；服务端嵌入方必须自行施加出口边界。
