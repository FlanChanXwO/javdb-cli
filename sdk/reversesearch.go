package javdb

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/provider"
)

// ReverseSearchSource 描述一个反搜 source。Name 为空或 "builtin" 时使用内置
// AVScan provider；否则 URL 必须是绝对 HTTP(S) 地址，Headers 必须已展开。
type ReverseSearchSource struct {
	Name    string
	URL     string
	Headers map[string]string
}

// ReverseSearchRequest 是一次以图搜番的原始图片输入。
type ReverseSearchRequest struct {
	Image    []byte
	Filename string
	Source   ReverseSearchSource
	// BypassCache 请求级绕过缓存读写。
	BypassCache bool
}

// ReverseSearchFrame 是候选结果中的一帧。
type ReverseSearchFrame struct {
	ImageName    string
	Similarity   float64
	Timestamp    string
	ThumbnailURL string
}

// ReverseSearchCandidate 是一个视频候选。
type ReverseSearchCandidate struct {
	VideoCode  string
	Similarity float64
	Frames     []ReverseSearchFrame
}

// ReverseSearchResponse 是反搜的规范化结果。
type ReverseSearchResponse struct {
	Source     string
	Candidates []ReverseSearchCandidate
}

// ReverseSearchCache 是 SDK 可注入的响应缓存。key 是原图 SHA-256 十六进制；
// 实现不得保存图片、鉴权 header 或 JavDB 详情。CLI 以本接口注入本机文件缓存。
type ReverseSearchCache interface {
	Get(context.Context, string) (ReverseSearchResponse, bool, error)
	Put(context.Context, string, ReverseSearchResponse) error
}

// ReverseSearchOptions 配置反搜传输与缓存。零值使用默认三次总请求与
// 30/60 秒退避、60 秒单请求超时。
type ReverseSearchOptions struct {
	Cache          ReverseSearchCache
	Retries        int
	RetryWait      time.Duration
	RequestTimeout time.Duration
}

// ImageSearchOptions 控制候选到 JavDB 的联动解析；当前固定严格精确匹配，
// 保留该类型以便未来扩展（并发数、解析重试等）。
type ImageSearchOptions struct{}

// ImageSearchError 是单候选联动失败的稳定错误。
type ImageSearchError struct {
	Stage   string
	Code    string
	Message string
}

func (e *ImageSearchError) Error() string {
	if e == nil {
		return "image search error"
	}
	return e.Stage + ": " + e.Code + ": " + e.Message
}

// ImageSearchMatch 是单个候选的联动结果；失败时 Error 非空。
type ImageSearchMatch struct {
	Candidate ReverseSearchCandidate
	MovieID   string
	Movie     map[string]any
	Error     *ImageSearchError
}

// ImageSearchResult 保留 provider 原始候选顺序。
type ImageSearchResult struct {
	ReverseSearch ReverseSearchResponse
	Matches       []ImageSearchMatch
}

// WithReverseSearch 注入反搜缓存与传输配置。javdb.New 本身仍不联网。
func WithReverseSearch(opts ReverseSearchOptions) Option {
	return func(o *options) { o.reverseSearch = opts }
}

// ReverseSearch 上传原始图片并返回规范化候选。Source 为空时使用内置
// AVScan provider。provider 顶层失败返回 error；缓存命中时跳过 provider。
func (c *Client) ReverseSearch(ctx context.Context, request ReverseSearchRequest) (ReverseSearchResponse, error) {
	if c == nil || c.api == nil {
		return ReverseSearchResponse{}, errNilClient()
	}
	if len(request.Image) == 0 {
		return ReverseSearchResponse{}, errEmptyImage()
	}
	if c.reverseSearch.Cache != nil && !request.BypassCache {
		key := imageCacheKey(request.Image)
		cached, hit, err := c.reverseSearch.Cache.Get(ctx, key)
		if err != nil {
			return ReverseSearchResponse{}, err
		}
		if hit {
			return cached, nil
		}
	}

	httpClient, err := c.reverseHTTPClient()
	if err != nil {
		return ReverseSearchResponse{}, err
	}
	providerOptions := provider.Options{
		HTTPClient:     httpClient,
		RequestTimeout: c.reverseSearch.RequestTimeout,
		Retries:        c.reverseSearch.Retries,
		RetryWait:      c.reverseSearch.RetryWait,
	}
	var selected provider.Provider
	if request.Source.Name == "" || request.Source.Name == provider.BuiltinName {
		selected = provider.NewBuiltin(providerOptions)
	} else {
		selected, err = provider.New(provider.Source{
			Name:    request.Source.Name,
			URL:     request.Source.URL,
			Headers: request.Source.Headers,
		}, providerOptions)
		if err != nil {
			return ReverseSearchResponse{}, err
		}
	}
	response, err := selected.Search(ctx, provider.Request{Image: request.Image, Filename: request.Filename})
	if err != nil {
		return ReverseSearchResponse{}, err
	}
	normalized := mapReverseSearchResponse(response)
	if c.reverseSearch.Cache != nil && !request.BypassCache {
		if err := c.reverseSearch.Cache.Put(ctx, imageCacheKey(request.Image), normalized); err != nil {
			return ReverseSearchResponse{}, err
		}
	}
	return normalized, nil
}

// SearchByImage 反搜并对全部候选并发执行严格番号解析与完整详情；结果按
// provider 原始顺序恢复。候选级失败写入 ImageSearchMatch.Error 并继续，
// provider 顶层失败直接返回 error。
func (c *Client) SearchByImage(ctx context.Context, request ReverseSearchRequest, _ ImageSearchOptions) (ImageSearchResult, error) {
	response, err := c.ReverseSearch(ctx, request)
	if err != nil {
		return ImageSearchResult{}, err
	}
	result := ImageSearchResult{ReverseSearch: response}
	matches := make([]ImageSearchMatch, len(response.Candidates))
	if len(response.Candidates) == 0 {
		return result, nil
	}
	var wait sync.WaitGroup
	wait.Add(len(response.Candidates))
	for index, candidate := range response.Candidates {
		go func(index int, candidate ReverseSearchCandidate) {
			defer wait.Done()
			match := ImageSearchMatch{Candidate: candidate}
			movieID, err := c.ResolveMovieIDExact(ctx, candidate.VideoCode)
			if err != nil {
				match.Error = &ImageSearchError{Stage: "link", Code: "resolve", Message: err.Error()}
				matches[index] = match
				return
			}
			movie, err := c.MovieDetail(ctx, movieID)
			if err != nil {
				match.Error = &ImageSearchError{Stage: "link", Code: "detail", Message: err.Error()}
				match.MovieID = movieID
				matches[index] = match
				return
			}
			match.MovieID = movieID
			match.Movie = movie
			matches[index] = match
		}(index, candidate)
	}
	wait.Wait()
	result.Matches = matches
	return result, nil
}

// ResolveMovieIDExact 使用 zone=all 搜索，只接受大小写不敏感的完整相等番号；
// 零匹配与多重精确匹配都显式失败，不沿用旧 ResolveMovieID 的首项回退语义。
func (c *Client) ResolveMovieIDExact(ctx context.Context, number string) (string, error) {
	if c == nil || c.api == nil {
		return "", errNilClient()
	}
	return c.api.ResolveMovieIDExact(ctx, number)
}

func imageCacheKey(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func errNilClient() error {
	return errors.New("javdb client is nil")
}

func errEmptyImage() error {
	return errors.New("reverse search image bytes are empty")
}

func mapReverseSearchResponse(response *provider.Response) ReverseSearchResponse {
	normalized := ReverseSearchResponse{Source: response.Source}
	for _, candidate := range response.Candidates {
		mapped := ReverseSearchCandidate{VideoCode: candidate.VideoCode, Similarity: candidate.Similarity}
		for _, frame := range candidate.Frames {
			mapped.Frames = append(mapped.Frames, ReverseSearchFrame{
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
