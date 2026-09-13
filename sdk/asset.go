package javdb

import (
	"context"
	"fmt"
)

// 资产类型常量。资产领域只有这两种媒体类型。
const (
	assetTypeImage = "image"
	assetTypeVideo = "video"
)

// MovieAsset 是影片可下载媒体资产的最小描述。
// Type 只有 "image" 或 "video";URL 是资产的获取地址。
// 刻意不包含 id/index/role 等元数据:编号只是 list 输出的位置,描述只是 TTY 渲染。
type MovieAsset struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// MovieAssets 按固定顺序返回影片详情中的资产序列:
// thumbnail → cover(若详情提供)→ preview_images[](large_url 优先,回退 thumb_url)→ preview video。
// 详情中缺失的项直接跳过,因此序列长度随影片而变。
func (c *Client) MovieAssets(ctx context.Context, movieID string) ([]MovieAsset, error) {
	_ = ctx
	movie, err := c.api.MovieDetail(movieID)
	if err != nil {
		return nil, fmt.Errorf("fetch movie detail: %w", err)
	}
	return movieAssetsFromDetail(movie), nil
}

// movieAssetsFromDetail 把影片详情 map 转成资产序列。
// 详情无 typed struct(上层 API 返回 map[string]any),字段缺失/类型异常一律跳过。
func movieAssetsFromDetail(movie map[string]any) []MovieAsset {
	assets := make([]MovieAsset, 0)
	if url := movieString(movie["thumb_url"]); url != "" {
		assets = append(assets, MovieAsset{Type: assetTypeImage, URL: url})
	}
	if url := movieString(movie["cover_url"]); url != "" {
		assets = append(assets, MovieAsset{Type: assetTypeImage, URL: url})
	}
	if url := moviePreviewImageURLs(movie); len(url) > 0 {
		for _, preview := range url {
			assets = append(assets, MovieAsset{Type: assetTypeImage, URL: preview})
		}
	}
	if url := movieString(movie["preview_video_url"]); url != "" {
		assets = append(assets, MovieAsset{Type: assetTypeVideo, URL: url})
	}
	return assets
}

// moviePreviewImageURLs 展开全部 preview_images 的展示 URL。
// 同一张图内 large_url 与 thumb_url 是同一资产的不同来源,只产出一个 URL。
func moviePreviewImageURLs(movie map[string]any) []string {
	items, ok := movie["preview_images"].([]any)
	if !ok {
		return nil
	}
	urls := make([]string, 0, len(items))
	for _, item := range items {
		preview, ok := item.(map[string]any)
		if !ok {
			continue
		}
		url := movieString(preview["large_url"])
		if url == "" {
			url = movieString(preview["thumb_url"])
		}
		if url == "" {
			continue
		}
		urls = append(urls, url)
	}
	return urls
}
