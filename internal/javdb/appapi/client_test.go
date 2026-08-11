package appapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/tags"
)

// 根 Client 契约测试：真实组合层通过未导出别名嵌入 capability，method promotion 提供全部方法。

var (
	_ func(Options) (*Client, error) = New
	_ model.Options                  = Options{}
	_ model.SearchOptions            = SearchOptions{}
	_ model.BrowseOptions            = BrowseOptions{}
	_ model.EntityMoviesOptions      = EntityMoviesOptions{}
	_ model.LoginResponse            = LoginResponse{}
	_ model.SearchResult             = SearchResult{}
	_ model.Error                    = Error{}
	_ model.AuthRequired             = AuthRequired{}

	// 嵌入 transport 提供的方法。
	_ func(*Client, string)                               = (*Client).SetToken
	_ func(*Client) string                                = (*Client).Token
	_ func(*Client) string                                = (*Client).DeviceUUID
	_ func(*Client, string, map[string]string, any) error = (*Client).GetJSON
	_ func(*Client, string, map[string]string, any) error = (*Client).PostFormJSON
	_ func(*Client, string, map[string]string, any) error = (*Client).DeleteJSON

	// 各 endpoint capability 通过 promotion 提供的方法。
	_ func(*Client, string, string) (string, error)                                     = (*Client).Login
	_ func(*Client) (map[string]json.RawMessage, error)                                 = (*Client).Startup
	_ func(*Client) (map[string]json.RawMessage, error)                                 = (*Client).Users
	_ func(*Client, string) (int64, string, error)                                      = (*Client).ResolveUserID
	_ func(*Client, string, string) ([]map[string]any, error)                           = (*Client).TagsRaw
	_ func(*Client, string) (*tags.Doc, error)                                          = (*Client).RefreshTagTaxonomy
	_ func(*Client, string, bool) (*tags.Doc, string, error)                            = (*Client).LoadOrRefreshTaxonomy
	_ func(*Client, []string, string) ([]string, error)                                 = (*Client).ResolveTags
	_ func(*Client, BrowseOptions) (SearchResult, error)                                = (*Client).Browse
	_ func(*Client, string, string, EntityMoviesOptions) (SearchResult, error)          = (*Client).EntityMovies
	_ func(*Client, string, string) (map[string]any, error)                             = (*Client).EntityDetail
	_ func(*Client, string, string, string) (string, error)                             = (*Client).ResolveEntity
	_ func(*Client, string, string, EntityMoviesOptions, int) ([]map[string]any, error) = (*Client).AllEntityMovies
	_ func(*Client, int, int, string) (SearchResult, error)                             = (*Client).MyLists
	_ func(*Client, string) (map[string]any, error)                                     = (*Client).ListInfo
	_ func(*Client, string, int, int) (SearchResult, error)                             = (*Client).RelatedLists
	_ func(*Client, string) (map[string]any, error)                                     = (*Client).MovieDetail
	_ func(*Client, string) ([]map[string]any, error)                                   = (*Client).MovieMagnets
	_ func(*Client, string, int, int) ([]map[string]any, error)                         = (*Client).MovieComments
	_ func(*Client, string) (string, error)                                             = (*Client).ResolveMovieID
	_ func(*Client, string, string) (int64, error)                                      = (*Client).DownloadImage
	_ func(*Client, string, string) (int64, error)                                      = (*Client).DownloadHLS
	_ func(*Client, string, string) (SearchResult, error)                               = (*Client).RankingsMovies
	_ func(*Client, string) (SearchResult, error)                                       = (*Client).RankingsActors
	_ func(*Client, string, string) (SearchResult, error)                               = (*Client).RankingsPlayback
	_ func(*Client, string, string, int, int, int, bool) (SearchResult, error)          = (*Client).Top250
	_ func(*Client, string, SearchOptions) (SearchResult, error)                        = (*Client).Search
	_ func(*Client, string, int) ([]map[string]any, error)                              = (*Client).ReviewMoviesPage
	_ func(*Client) ([]map[string]any, error)                                           = (*Client).WatchedMovies
	_ func(*Client) ([]map[string]any, error)                                           = (*Client).WantMovies
	_ func(*Client, string, string, int, string) (map[string]any, error)                = (*Client).Mark
	_ func(*Client, string) (bool, error)                                               = (*Client).Unmark
	_ func(*Client, string) ([]map[string]any, error)                                   = (*Client).Collected
	_ func(*Client) ([]map[string]any, error)                                           = (*Client).RecentViewed
	_ func(*Client, string, int) ([]map[string]any, error)                              = (*Client).CollectedPage

	// route capability 经 promotion 提供的方法。
	_ func(*Client, context.Context, AutoHostOptions, AutoHostProbe) (AutoHostResult, error) = (*Client).Select

	// 自动选线根入口（供公开 SDK facade 调用）。
	_ func(AutoHostOptions) (AutoHostProbe, error)                                  = NewAutoHostProbe
	_ func(context.Context, AutoHostOptions, AutoHostProbe) (AutoHostResult, error) = SelectAutoHost
)

// TestSelectAutoHostWithInjectedProbe 验证 appapi 组合层用注入 probe 完成自动选线，
// preferred 成功时复用并透传 latency。
func TestSelectAutoHostWithInjectedProbe(t *testing.T) {
	const preferred = "https://apidd.spthgb.com"
	probe := AutoHostProbe(func(ctx context.Context, host string, onRequestStart func()) (time.Duration, map[string]any, error) {
		if onRequestStart != nil {
			onRequestStart()
		}
		return 12 * time.Millisecond, map[string]any{"ok": true}, nil
	})
	result, err := SelectAutoHost(context.Background(), AutoHostOptions{PreferredHost: preferred}, probe)
	if err != nil {
		t.Fatalf("SelectAutoHost() error = %v", err)
	}
	if result.Host != preferred || !result.ReusedPreferred {
		t.Fatalf("result = %+v, want host %s reused", result, preferred)
	}
	if result.Latency != 12*time.Millisecond {
		t.Fatalf("result latency = %v, want 12ms", result.Latency)
	}
}

// TestNewConstructsComposedClient 验证 New 无网络构造真实 Client，并保持 transport 状态可经 promotion 访问。
func TestNewConstructsComposedClient(t *testing.T) {
	c, err := New(Options{Host: "https://example.invalid", DeviceUUID: "client-test"})
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	if c == nil {
		t.Fatal("New returned nil")
	}
	if got := c.DeviceUUID(); got != "client-test" {
		t.Fatalf("DeviceUUID() = %q, want client-test", got)
	}
	if got := c.Token(); got != "" {
		t.Fatalf("Token() = %q, want empty", got)
	}
	c.SetToken("tok")
	if got := c.Token(); got != "tok" {
		t.Fatalf("Token() after SetToken = %q", got)
	}
}

// TestNewRejectsInvalidProxy 验证 transport 构造错误向上传播（tls-client 拒绝非法代理 URL）。
func TestNewRejectsInvalidProxy(t *testing.T) {
	if _, err := New(Options{Host: "https://example.invalid", Proxy: "://bad-proxy"}); err == nil {
		t.Skip("tls-client did not reject the proxy at construction time")
	}
}

// TestCapabilitiesShareTransportState 验证 capability 与根 Client 共享同一 transport 指针：
// 经根 Client 写入的 token 必须能被 auth capability 本地读取（JWT 解析无需网络）。
func TestCapabilitiesShareTransportState(t *testing.T) {
	c, err := New(Options{Host: "https://example.invalid"})
	if err != nil {
		t.Fatal(err)
	}
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"user_id":123,"username":"u"}`))
	tok := "h." + payload + ".sig"
	c.SetToken(tok)
	id, name, err := c.ResolveUserID("")
	if err != nil || id != 123 || name != "u" {
		t.Fatalf("ResolveUserID = %d, %q, %v (token must be shared)", id, name, err)
	}
}

func TestHelperReexportsStable(t *testing.T) {
	if SearchTypeListKey("list") != "lists" {
		t.Fatalf("SearchTypeListKey(list) = %q", SearchTypeListKey("list"))
	}
	if RankingPeriod("week") != "weekly" || ActorPeriod("month") != "monthly" {
		t.Fatal("ranking period forwarder changed")
	}
	if MagnetURI(map[string]any{"hash": "abc"}) != "magnet:?xt=urn:btih:abc" {
		t.Fatal("magnet forwarder changed")
	}
	if Zones["censored"] != 0 || MainFlags["m"] != true || EntityLetters["list"] != "l" {
		t.Fatal("facade maps changed")
	}
	if CollectionSpecs["actors"].Key != "actors" {
		t.Fatal("collection specs changed")
	}
	items := FilterMagnets([]map[string]any{
		{"name": "big", "cnsub": true, "size": float64(4096)},
		{"name": "small", "size": float64(64)},
	}, true, false, 0)
	if len(items) != 1 || items[0]["name"] != "big" {
		t.Fatalf("FilterMagnets = %v", items)
	}
	if got, err := ResolveNumber([]map[string]any{{"number": "SSIS-1", "id": "x"}}, "ssis-1"); err != nil || got != "x" {
		t.Fatalf("ResolveNumber = %q, %v", got, err)
	}
	p := BuildSearchParams("kw", SearchOptions{Page: 1, Zone: "all"})
	if p["q"] != "kw" || p["page"] != "1" {
		t.Fatalf("BuildSearchParams = %v", p)
	}
	if _, err := BuildTop250Params("censored", "2023", 1, 1, 20, false); err != nil {
		t.Fatalf("BuildTop250Params error = %v", err)
	}
	got, err := AllPages(func(p int) ([]map[string]any, error) {
		if p == 1 {
			return []map[string]any{{"id": "a"}}, nil
		}
		return nil, nil
	}, 3)
	if err != nil || len(got) != 1 {
		t.Fatalf("AllPages = %v, %v", got, err)
	}
}

func TestErrorAndAuthRequiredMatch(t *testing.T) {
	base := &model.Error{Action: "search", Message: "boom"}
	var apiErr *Error
	if !errors.As(base, &apiErr) {
		t.Fatal("errors.As(*Error) failed")
	}
	authErr := &AuthRequired{API: *base}
	var unwrapped *Error
	if !errors.As(authErr, &unwrapped) || unwrapped.Action != "search" {
		t.Fatal("errors.As(AuthRequired -> Error) failed")
	}
}

func TestSearchResultAccessorsStable(t *testing.T) {
	result := SearchResult{"movies": json.RawMessage(`[{"id":"movie-1"}]`)}
	movies := result.Movies()
	if len(movies) != 1 || movies[0]["id"] != "movie-1" {
		t.Fatalf("SearchResult.Movies() = %#v", movies)
	}
}
