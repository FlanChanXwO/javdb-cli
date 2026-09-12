package javdb

import (
	"context"
	"fmt"
)

// MovieAssetDownloadOptions 指定要保存到本地的影片资源。空路径表示不下载该资源。
// PreviewImagePath 始终只对应详情返回的第一张预览图。
type MovieAssetDownloadOptions struct {
	ThumbnailPath    string
	PreviewImagePath string
	PreviewVideoPath string
}

// MovieAssetDownloadResult 记录实际写入的资源路径和字节数。
type MovieAssetDownloadResult struct {
	ThumbnailPath     string
	ThumbnailBytes    int64
	PreviewImagePath  string
	PreviewImageBytes int64
	PreviewVideoPath  string
	PreviewVideoBytes int64
}

// MovieDetail returns GET /api/v4/movies/{id} nested movie map.
func (c *Client) MovieDetail(ctx context.Context, movieID string) (map[string]any, error) {
	_ = ctx
	return c.api.MovieDetail(movieID)
}

// MovieMagnets returns GET /api/v1/movies/{id}/magnets (works anonymously).
func (c *Client) MovieMagnets(ctx context.Context, movieID string) ([]map[string]any, error) {
	_ = ctx
	return c.api.MovieMagnets(movieID)
}

// MovieComments returns one requested page from GET /api/v1/movies/{id}/reviews.
// It never follows later pages automatically.
func (c *Client) MovieComments(ctx context.Context, movieID string, page, limit int) ([]map[string]any, error) {
	_ = ctx
	return c.api.MovieComments(movieID, page, limit)
}

// DownloadMovieAssets 下载调用方选择的影片资源。图片会先验证并还原为标准图片字节；
// 预览视频会完整下载已结束的 HLS 预览流。输出文件不得预先存在。
func (c *Client) DownloadMovieAssets(ctx context.Context, movieID string, opt MovieAssetDownloadOptions) (MovieAssetDownloadResult, error) {
	_ = ctx
	if opt.ThumbnailPath == "" && opt.PreviewImagePath == "" && opt.PreviewVideoPath == "" {
		return MovieAssetDownloadResult{}, fmt.Errorf("select at least one movie asset output")
	}
	if err := distinctMovieAssetPaths(opt); err != nil {
		return MovieAssetDownloadResult{}, err
	}

	movie, err := c.api.MovieDetail(movieID)
	if err != nil {
		return MovieAssetDownloadResult{}, fmt.Errorf("fetch movie detail: %w", err)
	}
	sources := movieAssetURLs(movie)
	result := MovieAssetDownloadResult{}
	if opt.ThumbnailPath != "" {
		if sources.thumbnail == "" {
			return result, fmt.Errorf("movie has no thumbnail")
		}
		result.ThumbnailBytes, err = c.api.DownloadImage(sources.thumbnail, opt.ThumbnailPath)
		if err != nil {
			return result, fmt.Errorf("download thumbnail: %w", err)
		}
		result.ThumbnailPath = opt.ThumbnailPath
	}
	if opt.PreviewImagePath != "" {
		if sources.previewImage == "" {
			return result, fmt.Errorf("movie has no first preview image")
		}
		result.PreviewImageBytes, err = c.api.DownloadImage(sources.previewImage, opt.PreviewImagePath)
		if err != nil {
			return result, fmt.Errorf("download first preview image: %w", err)
		}
		result.PreviewImagePath = opt.PreviewImagePath
	}
	if opt.PreviewVideoPath != "" {
		if sources.previewVideo == "" {
			return result, fmt.Errorf("movie has no preview video")
		}
		result.PreviewVideoBytes, err = c.api.DownloadHLS(sources.previewVideo, opt.PreviewVideoPath)
		if err != nil {
			return result, fmt.Errorf("download preview video: %w", err)
		}
		result.PreviewVideoPath = opt.PreviewVideoPath
	}
	return result, nil
}

type movieAssetURLsResult struct {
	thumbnail    string
	previewImage string
	previewVideo string
}

func movieAssetURLs(movie map[string]any) movieAssetURLsResult {
	result := movieAssetURLsResult{
		thumbnail:    movieString(movie["thumb_url"]),
		previewVideo: movieString(movie["preview_video_url"]),
	}
	// 用户选择“预览图”时只取 index 0；不跳过无效首项，也不枚举其他图片。
	if preview := firstMovieMap(movie["preview_images"]); preview != nil {
		result.previewImage = movieString(preview["large_url"])
		if result.previewImage == "" {
			result.previewImage = movieString(preview["thumb_url"])
		}
	}
	return result
}

func firstMovieMap(value any) map[string]any {
	switch items := value.(type) {
	case []any:
		if len(items) == 0 {
			return nil
		}
		item, _ := items[0].(map[string]any)
		return item
	case []map[string]any:
		if len(items) == 0 {
			return nil
		}
		return items[0]
	default:
		return nil
	}
}

func movieString(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func distinctMovieAssetPaths(opt MovieAssetDownloadOptions) error {
	seen := map[string]bool{}
	for _, path := range []string{opt.ThumbnailPath, opt.PreviewImagePath, opt.PreviewVideoPath} {
		if path == "" {
			continue
		}
		if seen[path] {
			return fmt.Errorf("movie asset output paths must be distinct")
		}
		seen[path] = true
	}
	return nil
}

// ResolveMovieID maps a printed number to internal id (search zone=all).
func (c *Client) ResolveMovieID(ctx context.Context, number string) (string, error) {
	_ = ctx
	return c.api.ResolveMovieID(number)
}
