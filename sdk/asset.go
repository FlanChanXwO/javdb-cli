package javdb

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
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

// DownloadMovieAsset 下载单个影片资产到 target 路径,返回写入字节数。
// image:下载 → 必要时 XOR 解包 → 图片魔数校验 → 原子写入,不做任何格式转换。
// video:输出格式由 target 后缀决定——.ts 保留 MPEG-TS(解密后的完整流),
// 其余后缀(含 .mp4,remux 层落地前)一律拒绝,不做转码。
func (c *Client) DownloadMovieAsset(ctx context.Context, asset MovieAsset, target string) (int64, error) {
	_ = ctx
	switch asset.Type {
	case assetTypeImage:
		written, err := c.api.DownloadImage(asset.URL, target)
		if err != nil {
			return 0, fmt.Errorf("download image asset: %w", err)
		}
		return written, nil
	case assetTypeVideo:
		switch strings.ToLower(filepath.Ext(target)) {
		case ".ts":
			written, err := c.api.DownloadHLS(asset.URL, target)
			if err != nil {
				return 0, fmt.Errorf("download video asset: %w", err)
			}
			return written, nil
		default:
			// .mp4 在 TS→MP4 remux 实装前与未知后缀同路拒绝,文案与最终契约一致。
			return 0, fmt.Errorf("unsupported video output format %q", filepath.Ext(target))
		}
	default:
		return 0, fmt.Errorf("unsupported asset type %q", asset.Type)
	}
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
