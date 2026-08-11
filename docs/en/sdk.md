# javdb Go SDK

[Documentation](../index.md) · [简体中文](../zh-CN/sdk.md)

The public SDK is imported from `github.com/FlanChanXwO/javdb-cli/sdk` and
declares `package javdb`. It is the same remote capability surface used by the
CLI; it does not load `auth.json` or manage local accounts for an application.

## Install and construct

Pin an exact published tag in applications:

```bash
go get github.com/FlanChanXwO/javdb-cli/sdk@vX.Y.Z
```

## Migrating from `/javdb`

This is a breaking import-path migration. Replace
`github.com/FlanChanXwO/javdb-cli/javdb` with
`github.com/FlanChanXwO/javdb-cli/sdk` in each import declaration, then resolve
dependencies at a release containing this move. The package name, public types,
and documented methods remain `javdb`, so selectors such as `javdb.New` do not
change. The former `/javdb` import path is no longer supported.

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

Use `HostMain` or an explicit base URL only when the caller intends that route.
`WithTimeout` configures the HTTP client. For a stable device identity across
processes, a caller may use `javdb.LoadOrCreateDeviceUUID(path)` and pass the
returned value through `WithDeviceUUID`.

## Authentication

```go
ctx := context.Background()
token, err := client.Login(ctx, username, password)
if err != nil {
    return err
}
client.SetToken(token)
userID, username, err := client.ResolveUserID(ctx)
```

The caller owns credentials and persistence. Do not put a JWT or password in a
log, panic, error wrapper, or test fixture.

## Operations

| Capability | Methods |
| --- | --- |
| Discovery | `Search`, `MovieDetail`, `ResolveMovieID`, `Browse`, `ResolveTags` |
| Reviews | `MovieComments` |
| Local media downloads | `DownloadMovieMedia`, `MovieMediaDownloadOptions`, `MovieMediaDownloadResult` |
| Entity graph | `ResolveEntity`, `EntityDetail`, `EntityMovies`, `AllEntityMovies` |
| Magnets | `MovieMagnets`, `FilterMagnets`, `PickBestMagnet`, `MagnetURI` |
| Rankings | `RankingsMovies`, `RankingsActors`, `RankingsPlayback`, `Top250` |
| Personal state | `WatchedMovies`, `WantMovies`, `Mark`, `Unmark`, `Collected`, `RecentViewed` |
| Lists | `MyLists`, `ListInfo`, `RelatedLists` |
| Tag taxonomy | `RefreshTagTaxonomy`, `LoadOrRefreshTaxonomy` |

`RankingsMovies` and `RankingsPlayback` accept the zone names `censored`,
`uncensored`, `western`, and `fc2`, as well as an already numeric string. Known
names are normalized before the request. All three ranking methods accept
`day`, `week`, or `month`, as well as the API forms `daily`, `weekly`, or
`monthly`. `RankingPeriod` exposes the short-to-API mapping; `ActorPeriod`
remains as a deprecated compatibility alias.

Many list operations return `SearchResult`, with helpers for the response
dimension:

```go
result, err := client.Search(ctx, "SSIS", javdb.SearchOptions{
    Zone:  "censored",
    Limit: 10,
})
movies := result.Movies()
actors := result.Named("actors")
```

`MovieComments(ctx, movieID, page, limit)` requests exactly one page and never
walks subsequent pages. Non-positive values use page `1` and limit `20`, which
matches the CLI's one-page default.

Use `DownloadMovieMedia` only with explicitly chosen new local paths:

```go
downloaded, err := client.DownloadMovieMedia(ctx, movieID, javdb.MovieMediaDownloadOptions{
    PreviewImagePath: "/chosen/output/preview-0.jpg", // only preview_images[0]
    PreviewVideoPath: "/chosen/output/preview.ts",
})
if err != nil {
    return err
}
fmt.Println(downloaded.PreviewImageBytes, downloaded.PreviewVideoBytes)
```

Each non-empty path selects one media item. `PreviewImagePath` always uses only
the first preview image; the method never enumerates later images. Images are
validated before writing. The HLS video path supports completed single-media
playlists (including AES-128); master, byte-range, fragmented-MP4, and
unfinished/live playlists return an error. All selected outputs must be
distinct, their parent directories must exist, and no output may already exist.

Methods that update watch/want state or refresh the local public tag cache are
mutations. Call them only when the application has explicit authority to do so.
Media downloads are local file writes and likewise require an explicitly chosen
destination from the application user.

## Errors and compatibility

```go
var authRequired *javdb.AuthRequired
if errors.As(err, &authRequired) {
    // Re-authenticate through the caller's chosen credential flow.
}

var apiError *javdb.APIError
if errors.As(err, &apiError) {
    // The App API returned success:0 with a server-side failure.
}
```

The public package is the supported integration boundary. `internal/` paths,
wire payloads, signing details, and the `Client.API` escape hatch are not a
stable external contract. Pin a module version and use documented methods and
types in integrations.
