// Package result 提供 CLI 纯结果投影与过滤，按领域分文件（movie.go、magnet.go、named.go）。
//
// 类型与函数使用领域前缀避免含糊命名。包不接收 io.Writer、不含空列表文案、不编码 JSON、
// 不创建 Cobra command、不调用 SDK；只依赖 stdlib 与 internal/common/scalar。
package result

import (
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
)

// MovieRow 是影片列表单行的结构化投影。
type MovieRow struct {
	Number      string
	ID          string
	Title       string
	ReleaseDate string
}

// Line 返回影片行文本 `number\tid\ttitle[\trelease_date]`（不含尾随换行）。
func (r MovieRow) Line() string {
	line := r.Number + "\t" + r.ID + "\t" + r.Title
	if r.ReleaseDate != "" {
		line += "\t" + r.ReleaseDate
	}
	return line
}

// ProjectMovie 将影片记录投影为 MovieRow；保留 float64 ID 截断展示约定。
func ProjectMovie(item map[string]any) MovieRow {
	return MovieRow{
		Number:      display(item["number"]),
		ID:          display(item["id"]),
		Title:       display(item["title"]),
		ReleaseDate: display(item["release_date"]),
	}
}

// ProjectMovies 将影片记录列表投影为 MovieRow 列表，保持输入顺序。
func ProjectMovies(items []map[string]any) []MovieRow {
	out := make([]MovieRow, 0, len(items))
	for _, item := range items {
		out = append(out, ProjectMovie(item))
	}
	return out
}

// FilterMoviesWithMagnets 丢弃 magnets_count == 0 的行；缺失该字段的行保留。
func FilterMoviesWithMagnets(items []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if v, ok := item["magnets_count"]; ok {
			n := intValue(v)
			if n == 0 {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

// display 是 CLI 结果的字符串展示约定：浮点值截断为整数，其余委托 scalar。
func display(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return scalar.String(t)
	}
}

// intValue 是过滤与大小格式化的整数转换，保持浮点截断语义。
func intValue(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	default:
		if n, ok := scalar.Int64(v); ok {
			return int(n)
		}
		return 0
	}
}
