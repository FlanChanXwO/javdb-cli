package assets

import (
	"strings"
	"testing"

	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// movieString 只接受 string(计划 #10):非字符串字段不得 stringify 成
// "123"/"true"/"map[...]" 伪 URL 进入资产列表。
func TestMovieAssetsRejectsNonStringFields(t *testing.T) {
	movie := map[string]any{
		// 非字符串 thumb_url:必须跳过,不得 stringify。
		"thumb_url": 123,
		"cover_url": true,
		"preview_images": []any{
			map[string]any{"large_url": 456.0},
		},
		"preview_video_url": map[string]any{"nested": true},
	}
	assets := javdb.MovieAssetsFromDetail(movie)
	if len(assets) != 0 {
		t.Fatalf("non-string fields must not become assets, got %v", assets)
	}
}

// 字符串字段照常产出。
func TestMovieAssetsAcceptsStringFields(t *testing.T) {
	movie := map[string]any{
		"thumb_url": "https://example.test/thumb.jpg",
	}
	assets := javdb.MovieAssetsFromDetail(movie)
	if len(assets) != 1 || assets[0].URL != "https://example.test/thumb.jpg" {
		t.Fatalf("assets = %v", assets)
	}
}

// 空白字符串跳过。
func TestMovieAssetsSkipsBlankStrings(t *testing.T) {
	movie := map[string]any{
		"thumb_url": "   ",
		"cover_url": "https://example.test/cover.jpg",
	}
	assets := javdb.MovieAssetsFromDetail(movie)
	if len(assets) != 1 {
		t.Fatalf("assets = %v, want only cover", assets)
	}
	if !strings.Contains(assets[0].URL, "cover") {
		t.Fatalf("asset = %v", assets[0])
	}
}
