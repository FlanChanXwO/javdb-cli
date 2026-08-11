package javdb

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/tags"
)

// deprecated Client.API() 是冻结的兼容入口：返回 appapi 根 facade 且原方法集必须可编译、可调用。

var _ func(*Client) *appapi.Client = (*Client).API

// 原 appapi 根 facade 的方法集（API() 调用方可见）。任何目录重整不得改变这些签名。
var (
	_ func(c *appapi.Client, kind, entityID string, opt appapi.EntityMoviesOptions, maxPages int) ([]map[string]any, error)  = (*appapi.Client).AllEntityMovies
	_ func(c *appapi.Client, opt appapi.BrowseOptions) (appapi.SearchResult, error)                                          = (*appapi.Client).Browse
	_ func(c *appapi.Client, kind string) ([]map[string]any, error)                                                          = (*appapi.Client).Collected
	_ func(c *appapi.Client, kind string, page int) ([]map[string]any, error)                                                = (*appapi.Client).CollectedPage
	_ func(c *appapi.Client, path string, params map[string]string, dest any) error                                          = (*appapi.Client).DeleteJSON
	_ func(c *appapi.Client, playlistURL, target string) (int64, error)                                                      = (*appapi.Client).DownloadHLS
	_ func(c *appapi.Client, sourceURL, target string) (int64, error)                                                        = (*appapi.Client).DownloadImage
	_ func(c *appapi.Client, kind, id string) (map[string]any, error)                                                        = (*appapi.Client).EntityDetail
	_ func(c *appapi.Client, kind, entityID string, opt appapi.EntityMoviesOptions) (appapi.SearchResult, error)             = (*appapi.Client).EntityMovies
	_ func(c *appapi.Client, path string, params map[string]string, dest any) error                                          = (*appapi.Client).GetJSON
	_ func(c *appapi.Client, listID string) (map[string]any, error)                                                          = (*appapi.Client).ListInfo
	_ func(c *appapi.Client, zone string, force bool) (*tags.Doc, string, error)                                             = (*appapi.Client).LoadOrRefreshTaxonomy
	_ func(c *appapi.Client, username, password string) (string, error)                                                      = (*appapi.Client).Login
	_ func(c *appapi.Client, movieID, status string, score int, content string) (map[string]any, error)                      = (*appapi.Client).Mark
	_ func(c *appapi.Client, movieID string, page, limit int) ([]map[string]any, error)                                      = (*appapi.Client).MovieComments
	_ func(c *appapi.Client, movieID string) (map[string]any, error)                                                         = (*appapi.Client).MovieDetail
	_ func(c *appapi.Client, movieID string) ([]map[string]any, error)                                                       = (*appapi.Client).MovieMagnets
	_ func(c *appapi.Client, page, limit int, sortBy string) (appapi.SearchResult, error)                                    = (*appapi.Client).MyLists
	_ func(c *appapi.Client, path string, form map[string]string, dest any) error                                            = (*appapi.Client).PostFormJSON
	_ func(c *appapi.Client, period string) (appapi.SearchResult, error)                                                     = (*appapi.Client).RankingsActors
	_ func(c *appapi.Client, type_, period string) (appapi.SearchResult, error)                                              = (*appapi.Client).RankingsMovies
	_ func(c *appapi.Client, filterBy, period string) (appapi.SearchResult, error)                                           = (*appapi.Client).RankingsPlayback
	_ func(c *appapi.Client) ([]map[string]any, error)                                                                       = (*appapi.Client).RecentViewed
	_ func(c *appapi.Client, zone string) (*tags.Doc, error)                                                                 = (*appapi.Client).RefreshTagTaxonomy
	_ func(c *appapi.Client, movieID string, page, limit int) (appapi.SearchResult, error)                                   = (*appapi.Client).RelatedLists
	_ func(c *appapi.Client, kind, ref, zone string) (string, error)                                                         = (*appapi.Client).ResolveEntity
	_ func(c *appapi.Client, number string) (string, error)                                                                  = (*appapi.Client).ResolveMovieID
	_ func(c *appapi.Client, refs []string, zone string) ([]string, error)                                                   = (*appapi.Client).ResolveTags
	_ func(c *appapi.Client, token string) (int64, string, error)                                                            = (*appapi.Client).ResolveUserID
	_ func(c *appapi.Client, status string, page int) ([]map[string]any, error)                                              = (*appapi.Client).ReviewMoviesPage
	_ func(c *appapi.Client, keyword string, opt appapi.SearchOptions) (appapi.SearchResult, error)                          = (*appapi.Client).Search
	_ func(c *appapi.Client, token string)                                                                                   = (*appapi.Client).SetToken
	_ func(c *appapi.Client) (map[string]json.RawMessage, error)                                                             = (*appapi.Client).Startup
	_ func(c *appapi.Client, zone, lang string) ([]map[string]any, error)                                                    = (*appapi.Client).TagsRaw
	_ func(c *appapi.Client) string                                                                                          = (*appapi.Client).Token
	_ func(c *appapi.Client, zone, year string, startRank, page, limit int, ignoreWatched bool) (appapi.SearchResult, error) = (*appapi.Client).Top250
	_ func(c *appapi.Client, movieID string) (bool, error)                                                                   = (*appapi.Client).Unmark
	_ func(c *appapi.Client) (map[string]json.RawMessage, error)                                                             = (*appapi.Client).Users
	_ func(c *appapi.Client) ([]map[string]any, error)                                                                       = (*appapi.Client).WantMovies
	_ func(c *appapi.Client) ([]map[string]any, error)                                                                       = (*appapi.Client).WatchedMovies
	_ func(c *appapi.Client) string                                                                                          = (*appapi.Client).DeviceUUID
)

// taxonomy 返回类型是冻结兼容例外：RefreshTagTaxonomy 返回 *tags.Doc，
// LoadOrRefreshTaxonomy 返回 (*tags.Doc, string, error)。
// 上面方法值断言已锁定这两个签名，无需真实网络。

func TestClientAPIReturnsCompatibleFacade(t *testing.T) {
	client, err := New(
		WithHost("https://example.invalid"),
		WithDeviceUUID("device-test"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client.API() == nil {
		t.Fatal("API() returned nil")
	}
	if got := client.API().DeviceUUID(); got != "device-test" {
		t.Fatalf("API().DeviceUUID() = %q, want device-test", got)
	}
	if got := client.API().Token(); got != "" {
		t.Fatalf("API().Token() = %q, want empty", got)
	}
	client.API().SetToken("compat-token")
	if got := client.API().Token(); got != "compat-token" {
		t.Fatalf("API().Token() after SetToken = %q", got)
	}
}

func TestAPIErrorAndAuthRequiredMatchErrors(t *testing.T) {
	base := &appapi.Error{Action: "search", Message: "boom"}

	// errors.As 直接匹配 *APIError。
	var apiErr *APIError
	if !errors.As(base, &apiErr) {
		t.Fatalf("errors.As(*Error) failed for %T", base)
	}
	if apiErr.Action != "search" || apiErr.Message != "boom" {
		t.Fatalf("unwrapped APIError = %+v", apiErr)
	}

	// errors.As 匹配 *AuthRequired，并能沿 Unwrap 链路继续匹配底层 *APIError。
	authErr := &AuthRequired{API: *base}
	var matched *AuthRequired
	if !errors.As(authErr, &matched) {
		t.Fatalf("errors.As(AuthRequired) failed for %T", authErr)
	}
	var unwrapped *APIError
	if !errors.As(authErr, &unwrapped) {
		t.Fatalf("errors.As(AuthRequired -> APIError) failed")
	}
	if unwrapped.Action != "search" {
		t.Fatalf("unwrapped.Action = %q, want search", unwrapped.Action)
	}
	if got := authErr.Error(); got != "search: boom" {
		t.Fatalf("AuthRequired.Error() = %q, want %q", got, "search: boom")
	}
}

func TestDeprecatedActorPeriodAlias(t *testing.T) {
	// ActorPeriod 是 RankingPeriod 的兼容别名；行为必须一致。
	for _, period := range []string{"day", "week", "month", "daily", "weekly", "monthly", ""} {
		if got := ActorPeriod(period); got != RankingPeriod(period) {
			t.Fatalf("ActorPeriod(%q) = %q, RankingPeriod(%q) = %q", period, got, period, RankingPeriod(period))
		}
	}
	if got := ActorPeriod("day"); got != "daily" {
		t.Fatalf("ActorPeriod(day) = %q, want daily", got)
	}
}
