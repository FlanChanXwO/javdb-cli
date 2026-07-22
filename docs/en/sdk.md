# javdb Go SDK

[Documentation](../index.md) · [简体中文](../zh-CN/sdk.md)

The public package is `github.com/FlanChanXwO/javdb-cli/javdb`. It is the same
remote capability surface used by the CLI; it does not load `auth.json` or
manage local accounts for an application.

## Install and construct

Pin an exact published tag in applications:

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
| Entity graph | `ResolveEntity`, `EntityDetail`, `EntityMovies`, `AllEntityMovies` |
| Magnets | `MovieMagnets`, `FilterMagnets`, `PickBestMagnet`, `MagnetURI` |
| Rankings | `RankingsMovies`, `RankingsActors`, `RankingsPlayback`, `Top250` |
| Personal state | `WatchedMovies`, `WantMovies`, `Mark`, `Unmark`, `Collected`, `RecentViewed` |
| Lists | `MyLists`, `ListInfo`, `RelatedLists` |
| Tag taxonomy | `RefreshTagTaxonomy`, `LoadOrRefreshTaxonomy` |

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

Methods that update watch/want state or refresh the local public tag cache are
mutations. Call them only when the application has explicit authority to do so.

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
