// Package appapi 是签名 App API 的真实 Client 组合层。
//
// 根 Client 通过未导出类型别名嵌入 transport 与各 endpoint capability，使用 method
// promotion 提供公开 SDK `Client.API()` 当前可见的扁平方法集；不暴露可访问的 endpoint 字段，
// 不保留手写转发。具体协议实现位于 client、codec、media、model 与 endpoint 子包。
package appapi

import (
	"context"

	transport "github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/client"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/auth"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/browse"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/entity"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/lists"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/magnets"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/movie"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/rankings"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/route"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/search"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/user"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/media"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
)

const (
	// AppVersion 是请求使用的 App 版本字符串。
	AppVersion = model.AppVersion
	// AppVersionNumber 是请求使用的数值版 App 版本。
	AppVersionNumber = model.AppVersionNumber
	// UserAgent 是 App API 使用的 User-Agent。
	UserAgent = model.UserAgent
	// HostMirror 是默认的 JavDB API 镜像地址。
	HostMirror = model.HostMirror
	// HostMain 是 JavDB 主站地址。
	HostMain = model.HostMain
)

// 稳定类型通过 alias 从 model 暴露，保证 SDK、仓内调用方和根 Client 使用同一类型身份。
type (
	Options             = model.Options
	SearchOptions       = model.SearchOptions
	BrowseOptions       = model.BrowseOptions
	EntityMoviesOptions = model.EntityMoviesOptions
	LoginResponse       = model.LoginResponse
	SearchResult        = model.SearchResult
	Error               = model.Error
	AuthRequired        = model.AuthRequired
)

// 自动选线类型是 route capability 的 alias，供公开 SDK facade 在只导入根 appapi 的前提下使用。
type (
	AutoHostOptions = route.SelectorOptions
	AutoHostResult  = route.Result
	AutoHostProbe   = route.Probe
)

// 这些变量保留原根包入口，并与 model/endpoint 使用同一底层 map。
var (
	Zones           = model.Zones
	MainFlags       = browse.MainFlags
	EntityLetters   = browse.EntityLetters
	CollectionSpecs = user.CollectionSpecs
)

// 未导出指针类型别名：嵌入后字段名不可访问，但 method promotion 提供全部方法，
// 且所有 capability 共享同一 transport 指针（SetToken 等状态变更全局可见）。
type (
	transportClient = *transport.Client
	authClient      = *auth.AuthEndpoint
	browseClient    = *browse.BrowseEndpoint
	entityClient    = *entity.EntityEndpoint
	listsClient     = *lists.ListsEndpoint
	movieClient     = *movie.MovieEndpoint
	rankingsClient  = *rankings.RankingsEndpoint
	searchClient    = *search.SearchEndpoint
	userClient      = *user.UserEndpoint
	mediaClient     = *media.MediaEndpoint
	routeClient     = *route.RouteEndpoint
)

// Client 是真实组合层，嵌入 transport 与各 capability，仅构造一次 transport。
type Client struct {
	transportClient
	authClient
	browseClient
	entityClient
	listsClient
	movieClient
	rankingsClient
	searchClient
	userClient
	mediaClient
	routeClient
}

// New 按固定顺序构造 transport 与各 endpoint capability 并组合成根 Client。
func New(opts Options) (*Client, error) {
	t, err := transport.New(opts)
	if err != nil {
		return nil, err
	}
	a := auth.NewAuth(t)
	b := browse.NewBrowse(t)
	s := search.NewSearch(t)
	l := lists.NewLists(t)
	r := rankings.NewRankings(t)
	m := movie.NewMovie(t, s)
	e := entity.NewEntity(t, b, s)
	u := user.NewUser(t, m)
	md := media.NewMedia(t.FetchMedia)
	rt := route.NewRoute()
	return &Client{
		transportClient: t,
		authClient:      a,
		browseClient:    b,
		entityClient:    e,
		listsClient:     l,
		movieClient:     m,
		rankingsClient:  r,
		searchClient:    s,
		userClient:      u,
		mediaClient:     md,
		routeClient:     rt,
	}, nil
}

// AllPages 对分页回调聚合并按 id 去重。
func AllPages(fetch func(page int) ([]map[string]any, error), maxPages int) ([]map[string]any, error) {
	return user.AllPages(fetch, maxPages)
}

// BuildSearchParams 构建搜索 query 参数。
func BuildSearchParams(keyword string, opt SearchOptions) map[string]string {
	return search.BuildSearchParams(keyword, opt)
}

// BuildTagFilter 构建分类浏览 filter_by mask。
func BuildTagFilter(zone string, main, tags []string, year, month string) (string, error) {
	return browse.BuildTagFilter(zone, main, tags, year, month)
}

// BuildEntityFilter 构建实体作品 filter_by mask。
func BuildEntityFilter(kind, entityID, zone string, main []string) (string, error) {
	return browse.BuildEntityFilter(kind, entityID, zone, main)
}

// SearchTypeListKey 将实体类型映射到搜索响应 key。
func SearchTypeListKey(kind string) string { return entity.SearchTypeListKey(kind) }

// ResolveNumber 在搜索结果中解析番号。
func ResolveNumber(movies []map[string]any, number string) (string, error) {
	return movie.ResolveNumber(movies, number)
}

// RankingPeriod 将 CLI 周期映射为 API 周期。
func RankingPeriod(period string) string { return rankings.RankingPeriod(period) }

// ActorPeriod 是 RankingPeriod 的兼容别名。
func ActorPeriod(period string) string { return rankings.ActorPeriod(period) }

// BuildTop250Params 构建 TOP250 query 参数。
func BuildTop250Params(zone, year string, startRank, page, limit int, ignoreWatched bool) (map[string]string, error) {
	return rankings.BuildTop250Params(zone, year, startRank, page, limit, ignoreWatched)
}

// FilterMagnets 在本地按字幕、高清和大小过滤磁力。
func FilterMagnets(items []map[string]any, cnsub, hd bool, minSize int) []map[string]any {
	return magnets.FilterMagnets(items, cnsub, hd, minSize)
}

// PickBestMagnet 选择优先级最高的磁力。
func PickBestMagnet(items []map[string]any) map[string]any {
	return magnets.PickBestMagnet(items)
}

// MagnetURI 从磁力记录构建 magnet URI。
func MagnetURI(item map[string]any) string { return magnets.MagnetURI(item) }

// LoadOrCreateDeviceUUID 读取或创建稳定 device UUID。
func LoadOrCreateDeviceUUID(path string) (string, error) {
	return transport.LoadOrCreateDeviceUUID(path)
}

// NewAutoHostProbe 构造零重试的 /startup 自动选线探测函数。probe 共享同一 effective
// 网络选项与稳定 device UUID，不携带 bearer token。
func NewAutoHostProbe(opts AutoHostOptions) (AutoHostProbe, error) {
	return route.NewStartupProbe(opts)
}

// SelectAutoHost 执行并发动态线路选择，供公开 SDK facade 调用。probe 由调用方经
// NewAutoHostProbe 构造，便于注入可控探测。
func SelectAutoHost(ctx context.Context, opts AutoHostOptions, probe AutoHostProbe) (AutoHostResult, error) {
	return route.Select(ctx, opts, probe)
}
