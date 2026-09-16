package javdb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/media"
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
	movie, err := c.api.MovieDetailContext(ctx, movieID)
	if err != nil {
		return nil, fmt.Errorf("fetch movie detail: %w", err)
	}
	return MovieAssetsFromDetail(movie), nil
}

// DownloadMovieAsset 下载单个影片资产到 target 路径,返回写入字节数。
// image:下载 → 必要时 XOR 解包 → 图片魔数校验 → 原子写入,不做任何格式转换。
// video:输出格式由 target 后缀决定——.ts 保留解密校验后的 MPEG-TS,
// .mp4 输出 Fast Start MP4(纯 remux,不转码);其余后缀明确拒绝。
// ctx 贯穿全部阶段，取消时立即停止网络与媒体处理，不留输出文件。
func (c *Client) DownloadMovieAsset(ctx context.Context, asset MovieAsset, target string) (int64, error) {
	switch asset.Type {
	case assetTypeImage:
		written, err := c.api.DownloadImage(ctx, asset.URL, target)
		if err != nil {
			return 0, fmt.Errorf("download image asset: %w", err)
		}
		return written, nil
	case assetTypeVideo:
		switch strings.ToLower(filepath.Ext(target)) {
		case ".ts", ".mp4":
			written, err := c.api.DownloadHLS(ctx, asset.URL, target)
			if err != nil {
				return 0, fmt.Errorf("download video asset: %w", err)
			}
			return written, nil
		default:
			return 0, fmt.Errorf("unsupported video output format %q", filepath.Ext(target))
		}
	default:
		return 0, fmt.Errorf("unsupported asset type %q", asset.Type)
	}
}

// MovieAssetsFromDetail 把影片详情 map 转成资产序列(与 MovieAssetDescriptions 同序)。
// 详情无 typed struct(上层 API 返回 map[string]any),字段缺失/类型异常一律跳过。
func MovieAssetsFromDetail(movie map[string]any) []MovieAsset {
	assets, _ := movieAssetsWithDescriptions(movie)
	return assets
}

// MovieAssetDescriptions 返回与 MovieAssetsFromDetail 同序的 TTY 描述文本
// (thumbnail / cover / preview N / preview)。
// 描述仅供 assets list 的人类渲染,不属于 MovieAsset 数据模型,
// 不得进入 JSON/pipe 输出或任何持久化状态。跳过的项不占 preview 编号。
func MovieAssetDescriptions(movie map[string]any) []string {
	_, descs := movieAssetsWithDescriptions(movie)
	return descs
}

// ImageAssetFormat 读取本地图片文件的头部字节,返回稳定格式名
// (jpg/png/gif/webp/avif/heic),供 assets download 在下载后确定最终扩展名。
// 文件缺失或不是已验证的图片格式时返回 false。
func ImageAssetFormat(path string) (string, bool) {
	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	head := make([]byte, 64)
	n, err := file.Read(head)
	closeErr := file.Close()
	if closeErr != nil {
		return "", false
	}
	if err != nil && n == 0 {
		return "", false
	}
	return media.ImagePayloadFormat(head[:n])
}

// movieAssetsWithDescriptions 单次遍历详情 map,同序产出资产与其 TTY 描述。
func movieAssetsWithDescriptions(movie map[string]any) ([]MovieAsset, []string) {
	assets := make([]MovieAsset, 0)
	descs := make([]string, 0)
	add := func(assetType, url, desc string) {
		assets = append(assets, MovieAsset{Type: assetType, URL: url})
		descs = append(descs, desc)
	}
	if url := movieString(movie["thumb_url"]); url != "" {
		add(assetTypeImage, url, "thumbnail")
	}
	if url := movieString(movie["cover_url"]); url != "" {
		add(assetTypeImage, url, "cover")
	}
	// preview_images 内 large_url 与 thumb_url 是同一张图的不同来源,只产出一个 URL。
	preview := 0
	for _, url := range moviePreviewImageURLs(movie) {
		preview++
		add(assetTypeImage, url, fmt.Sprintf("preview %d", preview))
	}
	if url := movieString(movie["preview_video_url"]); url != "" {
		add(assetTypeVideo, url, "preview")
	}
	return assets, descs
}

// movieString 只接受 string 类型，非字符串字段不得
// stringify 成 "123"/"true"/"map[...]" 伪 URL 进入资产列表。
// TrimSpace 后非字符串/空白字符串直接跳过。
func movieString(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
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
