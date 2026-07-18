# Go SDK

[English](sdk.md) | [简体中文](sdk.zh-CN.md)

Public package: `github.com/FlanChanXwO/javdb-cli/javdb`

The SDK is a thin wrapper over the internal signed app API client. It does **not** read `auth.json` for you — pass tokens and call `Login` yourself (the CLI owns credential files).

## Install

```bash
go get github.com/FlanChanXwO/javdb-cli/javdb@latest
```

## Construct a client

```go
c, err := javdb.New(
    javdb.WithHost(javdb.HostMirror), // or HostMain / absolute URL
    javdb.WithProxy("http://127.0.0.1:7890"),
    javdb.WithToken(existingJWT),
    javdb.WithTimeout(20*time.Second),
    javdb.WithLang("en"),
    javdb.WithDeviceUUID(stableUUID),
)
```

## Auth

```go
ctx := context.Background()
token, err := c.Login(ctx, username, password)
uid, name, err := c.ResolveUserID(ctx)
c.SetToken(token)
```

`ResolveUserID` requires a successful login (or a valid token). It fails hard if no numeric user id can be resolved.

## Common methods

| Method | Purpose |
|--------|---------|
| `Search(ctx, q, SearchOptions)` | Keyword search |
| `MovieDetail(ctx, id)` | Movie map by internal id |
| `ResolveMovieID(ctx, number)` | 番号 → internal id |
| `MovieMagnets(ctx, id)` | Magnets (auth) |
| `FilterMagnets` / `PickBestMagnet` / `MagnetURI` | Client-side magnet helpers |
| `Browse(ctx, BrowseOptions)` | Category browse |
| `ResolveTags` / `LoadOrRefreshTaxonomy` | Tag aliases |
| `EntityMovies` / `ResolveEntity` / `EntityDetail` | Actor/series/…/list filmography |
| `WatchedMovies` / `WantMovies` / `Mark` / `Unmark` | User reviews |
| `Collected` / `RecentViewed` | Collections & recent |
| `RankingsMovies` / `RankingsActors` / `RankingsPlayback` | Rankings |
| `Top250` | TOP250 (auth) |
| `MyLists` / `ListInfo` / `RelatedLists` | 合集 |

Many list results use `SearchResult` (`map[string]json.RawMessage`) with helpers:

```go
res, err := c.Search(ctx, "SSIS", javdb.SearchOptions{Zone: "censored", Limit: 10})
movies := res.Movies()
// or res.Named("actors") for typed search dimensions
```

## Errors

```go
var ar *javdb.AuthRequired
if errors.As(err, &ar) {
    // token missing/expired — re-login
}
var api *javdb.APIError
if errors.As(err, &api) {
    // server success:0
}
```

## Notes

- Default host is the public mirror suitable for direct access.
- Thread-safety: one client per goroutine is safest (HTTP client is shared).
- Version of the CLI binary is unrelated to the SDK module version — use Go modules to pin.
