package javdb_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// 公开 SDK 契约测试（external test package）。
//
// 这些断言锁定本仓库公开 `sdk/` 的编译期与行为契约，任何目录重整都不得改变：
// 导出常量、函数签名、Client 方法签名、选项类型、别名类型身份以及纯 helper 行为。

// ---- 常量 ----
const _ = javdb.HostMirror
const _ = javdb.HostMain

// ---- 包级函数签名（编译期） ----
var (
	_ func(opts ...javdb.Option) (*javdb.Client, error)                      = javdb.New
	_ func(id string) javdb.Option                                           = javdb.WithDeviceUUID
	_ func(host string) javdb.Option                                         = javdb.WithHost
	_ func(lang string) javdb.Option                                         = javdb.WithLang
	_ func(proxy string) javdb.Option                                        = javdb.WithProxy
	_ func(d time.Duration) javdb.Option                                     = javdb.WithTimeout
	_ func(token string) javdb.Option                                        = javdb.WithToken
	_ func(path string) (string, error)                                      = javdb.LoadOrCreateDeviceUUID
	_ func(items []map[string]any, cnsub, hd bool, min int) []map[string]any = javdb.FilterMagnets
	_ func(items []map[string]any) map[string]any                            = javdb.PickBestMagnet
	_ func(items []map[string]any, count int) []map[string]any               = javdb.RankMagnets
	_ func(m map[string]any) string                                          = javdb.MagnetURI
	_ func(period string) string                                             = javdb.RankingPeriod
	_ func(period string) string                                             = javdb.ActorPeriod

	// 自动选线公开函数与类型（显式联网；javdb.New 保持无网络）。
	_ func(ctx context.Context, options javdb.AutoHostOptions) (javdb.AutoHostResult, error) = javdb.SelectAutoHost
	_ javdb.AutoHostOptions                                                                  = javdb.AutoHostOptions{}
	_ javdb.AutoHostResult                                                                   = javdb.AutoHostResult{}
)

// ---- Client 方法签名（编译期） ----
var (
	_ func(c *javdb.Client, ctx context.Context, kind, entityID string, opt javdb.EntityMoviesOptions, maxPages int) ([]map[string]any, error)  = (*javdb.Client).AllEntityMovies
	_ func(c *javdb.Client, ctx context.Context, opt javdb.BrowseOptions) (javdb.SearchResult, error)                                           = (*javdb.Client).Browse
	_ func(c *javdb.Client, ctx context.Context, kind string) ([]map[string]any, error)                                                         = (*javdb.Client).Collected
	_ func(c *javdb.Client, ctx context.Context, movieID string, opt javdb.MovieMediaDownloadOptions) (javdb.MovieMediaDownloadResult, error)   = (*javdb.Client).DownloadMovieMedia
	_ func(c *javdb.Client, ctx context.Context, kind, id string) (map[string]any, error)                                                       = (*javdb.Client).EntityDetail
	_ func(c *javdb.Client, ctx context.Context, kind, entityID string, opt javdb.EntityMoviesOptions) (javdb.SearchResult, error)              = (*javdb.Client).EntityMovies
	_ func(c *javdb.Client, ctx context.Context, listID string) (map[string]any, error)                                                         = (*javdb.Client).ListInfo
	_ func(c *javdb.Client, ctx context.Context, username, password string) (string, error)                                                     = (*javdb.Client).Login
	_ func(c *javdb.Client, ctx context.Context, movieID, status string, score int, content string) (map[string]any, error)                     = (*javdb.Client).Mark
	_ func(c *javdb.Client, ctx context.Context, movieID string, page, limit int) ([]map[string]any, error)                                     = (*javdb.Client).MovieComments
	_ func(c *javdb.Client, ctx context.Context, movieID string) (map[string]any, error)                                                        = (*javdb.Client).MovieDetail
	_ func(c *javdb.Client, ctx context.Context, movieID string) ([]map[string]any, error)                                                      = (*javdb.Client).MovieMagnets
	_ func(c *javdb.Client, ctx context.Context, page, limit int, sortBy string) (javdb.SearchResult, error)                                    = (*javdb.Client).MyLists
	_ func(c *javdb.Client, ctx context.Context, period string) (javdb.SearchResult, error)                                                     = (*javdb.Client).RankingsActors
	_ func(c *javdb.Client, ctx context.Context, type_, period string) (javdb.SearchResult, error)                                              = (*javdb.Client).RankingsMovies
	_ func(c *javdb.Client, ctx context.Context, filterBy, period string) (javdb.SearchResult, error)                                           = (*javdb.Client).RankingsPlayback
	_ func(c *javdb.Client, ctx context.Context) ([]map[string]any, error)                                                                      = (*javdb.Client).RecentViewed
	_ func(c *javdb.Client, ctx context.Context, movieID string, page, limit int) (javdb.SearchResult, error)                                   = (*javdb.Client).RelatedLists
	_ func(c *javdb.Client, ctx context.Context, kind, ref, zone string) (string, error)                                                        = (*javdb.Client).ResolveEntity
	_ func(c *javdb.Client, ctx context.Context, number string) (string, error)                                                                 = (*javdb.Client).ResolveMovieID
	_ func(c *javdb.Client, ctx context.Context, refs []string, zone string) ([]string, error)                                                  = (*javdb.Client).ResolveTags
	_ func(c *javdb.Client, ctx context.Context) (int64, string, error)                                                                         = (*javdb.Client).ResolveUserID
	_ func(c *javdb.Client, ctx context.Context, keyword string, opt javdb.SearchOptions) (javdb.SearchResult, error)                           = (*javdb.Client).Search
	_ func(c *javdb.Client, token string)                                                                                                       = (*javdb.Client).SetToken
	_ func(c *javdb.Client) string                                                                                                              = (*javdb.Client).Token
	_ func(c *javdb.Client, ctx context.Context, zone, year string, startRank, page, limit int, ignoreWatched bool) (javdb.SearchResult, error) = (*javdb.Client).Top250
	_ func(c *javdb.Client, ctx context.Context, movieID string) (bool, error)                                                                  = (*javdb.Client).Unmark
	_ func(c *javdb.Client, ctx context.Context) ([]map[string]any, error)                                                                      = (*javdb.Client).WantMovies
	_ func(c *javdb.Client, ctx context.Context) ([]map[string]any, error)                                                                      = (*javdb.Client).WatchedMovies
)

// ---- 别名类型身份 ----
// APIError/AuthRequired 是底层错误类型的别名，指针形式实现 error；SearchResult 是宽松 data map。
var (
	_ error                                                   = (*javdb.APIError)(nil)
	_ error                                                   = (*javdb.AuthRequired)(nil)
	_ javdb.SearchResult                                      = nil
	_ func(r javdb.SearchResult) []map[string]any             = javdb.SearchResult.Movies
	_ func(r javdb.SearchResult, key string) []map[string]any = javdb.SearchResult.Named
	_ javdb.BrowseOptions                                     = javdb.BrowseOptions{}
	_ javdb.EntityMoviesOptions                               = javdb.EntityMoviesOptions{}
	_ javdb.SearchOptions                                     = javdb.SearchOptions{}
	_ javdb.Option                                            = javdb.WithHost("mirror")
	_ javdb.MovieMediaDownloadOptions                         = javdb.MovieMediaDownloadOptions{}
	_ javdb.MovieMediaDownloadResult                          = javdb.MovieMediaDownloadResult{}
)

func TestExternalConstantsMatchLogicalHostNames(t *testing.T) {
	if javdb.HostMirror != "mirror" {
		t.Fatalf("HostMirror = %q, want mirror", javdb.HostMirror)
	}
	if javdb.HostMain != "main" {
		t.Fatalf("HostMain = %q, want main", javdb.HostMain)
	}
}

func TestExternalNewBuildsClientWithoutNetwork(t *testing.T) {
	client, err := javdb.New(
		javdb.WithHost(javdb.HostMirror),
		javdb.WithDeviceUUID("external-test-uuid"),
		javdb.WithToken("tok"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if client == nil {
		t.Fatal("New() returned nil client")
	}
	if got := client.Token(); got != "tok" {
		t.Fatalf("Token() = %q, want tok", got)
	}
	client.SetToken("tok2")
	if got := client.Token(); got != "tok2" {
		t.Fatalf("Token() after SetToken = %q, want tok2", got)
	}
	if api := client.API(); api == nil {
		t.Fatal("API() returned nil")
	}
}

// TestExternalSelectAutoHostWiringWithoutNetwork 验证公开 SelectAutoHost 已接入内部选线：
// 已取消的 context 会快速返回取消错误，不发真实网络请求、不悬挂。
func TestExternalSelectAutoHostWiringWithoutNetwork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := javdb.SelectAutoHost(ctx, javdb.AutoHostOptions{PreferredHost: "https://apidd.spthgb.com"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SelectAutoHost error = %v, want context.Canceled", err)
	}
}

func TestExternalMagnetHelpersPureBehavior(t *testing.T) {
	items := []map[string]any{
		{"name": "big", "size": float64(4096), "cnsub": true, "hash": "AAA"},
		{"name": "small", "size": float64(64), "hash": "BBB"},
	}
	filtered := javdb.FilterMagnets(items, true, false, 0)
	if len(filtered) != 1 || filtered[0]["name"] != "big" {
		t.Fatalf("FilterMagnets(cnsub) = %v", filtered)
	}
	best := javdb.PickBestMagnet(filtered)
	if best == nil || best["hash"] != "AAA" {
		t.Fatalf("PickBestMagnet = %v", best)
	}
	if got := javdb.MagnetURI(best); got != "magnet:?xt=urn:btih:AAA" {
		t.Fatalf("MagnetURI = %q", got)
	}
}

func TestExternalRankMagnetsStableSortAndTruncate(t *testing.T) {
	items := []map[string]any{
		{"name": "small", "size": float64(64), "hash": "BBB"},
		{"name": "big", "size": float64(4096), "cnsub": true, "hash": "AAA"},
		{"name": "mid", "size": float64(512), "hd": true, "hash": "CCC"},
	}
	// N=0: 全部排序，cnsub 优先 → big, hd 次之 → mid, 最后 small。
	all := javdb.RankMagnets(items, 0)
	if len(all) != 3 {
		t.Fatalf("RankMagnets(0) len = %d, want 3", len(all))
	}
	if all[0]["hash"] != "AAA" || all[1]["hash"] != "CCC" || all[2]["hash"] != "BBB" {
		t.Fatalf("RankMagnets(0) order = %+v", all)
	}
	// N=1: 截取第一个。
	one := javdb.RankMagnets(items, 1)
	if len(one) != 1 || one[0]["hash"] != "AAA" {
		t.Fatalf("RankMagnets(1) = %+v, want [AAA]", one)
	}
	// 不修改输入。
	if items[0]["name"] != "small" {
		t.Fatal("input slice was modified")
	}
}

func TestExternalPeriodMappings(t *testing.T) {
	// RankingPeriod/ActorPeriod 是 CLI 周期 -> API 周期的纯映射；两者行为一致。
	if got := javdb.RankingPeriod("day"); got != "daily" {
		t.Fatalf("RankingPeriod(day) = %q", got)
	}
	if got := javdb.ActorPeriod("weekly"); got != "weekly" {
		t.Fatalf("ActorPeriod(weekly) = %q", got)
	}
}

func TestExternalSearchResultAccessors(t *testing.T) {
	// 编译期已断言 Movies/Named 方法；这里确认空 payload 返回 nil 而非 panic。
	var raw javdb.SearchResult
	if got := raw.Movies(); got != nil {
		t.Fatalf("empty SearchResult.Movies() = %v", got)
	}
	if got := raw.Named("actors"); got != nil {
		t.Fatalf("empty SearchResult.Named() = %v", got)
	}
}

func TestExternalReflectTypeOfAPIResultIsPointer(t *testing.T) {
	client, err := javdb.New(javdb.WithHost("mirror"))
	if err != nil {
		t.Fatal(err)
	}
	got := reflect.TypeOf(client.API())
	if got == nil || got.Kind() != reflect.Ptr {
		t.Fatalf("API() reflect type = %v", got)
	}
}

// ---- 反搜公开契约（编译期） ----
var (
	_ func(opts javdb.ReverseSearchOptions) javdb.Option = javdb.WithReverseSearch

	_ javdb.ReverseSearchSource    = javdb.ReverseSearchSource{}
	_ javdb.ReverseSearchRequest   = javdb.ReverseSearchRequest{}
	_ javdb.ReverseSearchFrame     = javdb.ReverseSearchFrame{}
	_ javdb.ReverseSearchCandidate = javdb.ReverseSearchCandidate{}
	_ javdb.ReverseSearchResponse  = javdb.ReverseSearchResponse{}
	_ javdb.ReverseSearchOptions   = javdb.ReverseSearchOptions{}
	_ javdb.ImageSearchOptions     = javdb.ImageSearchOptions{}
	_ javdb.ImageSearchError       = javdb.ImageSearchError{}
	_ javdb.ImageSearchMatch       = javdb.ImageSearchMatch{}
	_ javdb.ImageSearchResult      = javdb.ImageSearchResult{}

	_ func(c *javdb.Client, ctx context.Context, req javdb.ReverseSearchRequest) (javdb.ReverseSearchResponse, error)                            = (*javdb.Client).ReverseSearch
	_ func(c *javdb.Client, ctx context.Context, req javdb.ReverseSearchRequest, opts javdb.ImageSearchOptions) (javdb.ImageSearchResult, error) = (*javdb.Client).SearchByImage
	_ func(c *javdb.Client, ctx context.Context, number string) (string, error)                                                                  = (*javdb.Client).ResolveMovieIDExact

	// 反搜缓存接口：CLI 以该接口注入本机文件缓存，SDK 不读取 ~/.javdb-cli。
	_ javdb.ReverseSearchCache = cacheAdapter{}
)

type cacheAdapter struct{}

func (cacheAdapter) Get(context.Context, string) (javdb.ReverseSearchResponse, bool, error) {
	return javdb.ReverseSearchResponse{}, false, nil
}

func (cacheAdapter) Put(context.Context, string, javdb.ReverseSearchResponse) error {
	return nil
}
