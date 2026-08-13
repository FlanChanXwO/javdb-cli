package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/cache"
	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/provider"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// ReverseSearchSetup 是一次反搜调用所需的公开 SDK client 与已解析 source。
type ReverseSearchSetup struct {
	Client *javdb.Client
	// Source 是本次调用选中的 source（已展开环境 header）；builtin 时
	// Name 为 "builtin"。
	Source javdb.ReverseSearchSource
	// CacheEnabled 反映配置与 --no-cache 之外的缓存开关。
	CacheEnabled bool
	// HTTPClient 是装配了最终代理配置的 HTTP client，供图片 URL 读取与
	// provider 请求共用（图片 URL、provider、JavDB 共用同一代理契约）。
	HTTPClient *http.Client
}

// NewReverseSearchClient 解析反搜配置（含环境 header 展开）、装配本机文件
// 缓存并构造携带反搜选项的公开 SDK client。explicitSource 非空时覆盖
// default_source。配置损坏显式失败，绝不偷回 builtin。
func NewReverseSearchClient(options *invocation.RootOptions, token, explicitSource string) (*ReverseSearchSetup, error) {
	rt, baseURL, err := resolveClient(options)
	if err != nil {
		return nil, err
	}
	path, err := paths.ConfigPath()
	if err != nil {
		return nil, err
	}
	file, err := settings.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load reverse search configuration: %w", err)
	}
	resolved, err := settings.ResolveReverseSearch(file, nil)
	if err != nil {
		return nil, err
	}

	sourceName := explicitSource
	if sourceName == "" {
		sourceName = resolved.DefaultSource
	}
	selected, err := selectResolvedSource(resolved, sourceName)
	if err != nil {
		return nil, err
	}

	var cacheAdapter javdb.ReverseSearchCache
	if resolved.Cache {
		cacheDir, err := paths.ReverseSearchCacheDir()
		if err != nil {
			return nil, err
		}
		store := cache.New(cacheDir, resolved.CacheTTL)
		cacheAdapter = sdkCacheAdapter{store: store, source: sourceName}
	}

	sdkClient, err := javdb.New(
		javdb.WithHost(baseURL),
		javdb.WithProxy(rt.Proxy),
		javdb.WithToken(token),
		javdb.WithDeviceUUID(rt.DeviceUUID),
		javdb.WithLang(rt.Lang),
		javdb.WithReverseSearch(javdb.ReverseSearchOptions{
			Cache:          cacheAdapter,
			Retries:        resolved.Retries,
			RetryWait:      resolved.RetryWait,
			RequestTimeout: resolved.RequestTimeout,
		}),
	)
	if err != nil {
		return nil, err
	}
	return &ReverseSearchSetup{
		Client:       sdkClient,
		Source:       selected,
		CacheEnabled: resolved.Cache,
		HTTPClient:   sdkClient.ReverseHTTPClient(),
	}, nil
}

// selectResolvedSource 返回 sourceName 对应的已展开 source；builtin 返回保留名。
func selectResolvedSource(resolved settings.ResolvedReverseSearch, sourceName string) (javdb.ReverseSearchSource, error) {
	if sourceName == provider.BuiltinName {
		return javdb.ReverseSearchSource{Name: provider.BuiltinName}, nil
	}
	for _, source := range resolved.Sources {
		if source.Name == sourceName {
			return javdb.ReverseSearchSource{
				Name:    source.Name,
				URL:     source.URL,
				Headers: source.Headers,
			}, nil
		}
	}
	return javdb.ReverseSearchSource{}, fmt.Errorf("reverse search source %q is not defined", sourceName)
}

// sdkCacheAdapter 把本机文件缓存适配为公开 SDK 缓存接口；source 由本次调用
// 固定，key 仍是原图 SHA-256。
type sdkCacheAdapter struct {
	store  *cache.Store
	source string
}

func (a sdkCacheAdapter) Get(ctx context.Context, key string) (javdb.ReverseSearchResponse, bool, error) {
	response, hit, err := a.store.Get(a.source, key)
	if err != nil || !hit {
		return javdb.ReverseSearchResponse{}, hit, err
	}
	return mapStoreResponse(response), true, nil
}

func (a sdkCacheAdapter) Put(ctx context.Context, key string, response javdb.ReverseSearchResponse) error {
	return a.store.Put(a.source, key, mapSDKResponse(response))
}

func mapStoreResponse(response *provider.Response) javdb.ReverseSearchResponse {
	normalized := javdb.ReverseSearchResponse{Source: response.Source}
	for _, candidate := range response.Candidates {
		mapped := javdb.ReverseSearchCandidate{VideoCode: candidate.VideoCode, Similarity: candidate.Similarity}
		for _, frame := range candidate.Frames {
			mapped.Frames = append(mapped.Frames, javdb.ReverseSearchFrame{
				ImageName:    frame.ImageName,
				Similarity:   frame.Similarity,
				Timestamp:    frame.Timestamp,
				ThumbnailURL: frame.ThumbnailURL,
			})
		}
		normalized.Candidates = append(normalized.Candidates, mapped)
	}
	return normalized
}

func mapSDKResponse(response javdb.ReverseSearchResponse) *provider.Response {
	normalized := &provider.Response{Source: response.Source}
	for _, candidate := range response.Candidates {
		mapped := provider.Candidate{VideoCode: candidate.VideoCode, Similarity: candidate.Similarity}
		for _, frame := range candidate.Frames {
			mapped.Frames = append(mapped.Frames, provider.Frame{
				ImageName:    frame.ImageName,
				Similarity:   frame.Similarity,
				Timestamp:    frame.Timestamp,
				ThumbnailURL: frame.ThumbnailURL,
			})
		}
		normalized.Candidates = append(normalized.Candidates, mapped)
	}
	return normalized
}
