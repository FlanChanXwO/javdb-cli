// Package movie 提供影片记录的纯投影与 has-magnets 过滤。
//
// 只处理影片记录字段投影与过滤；不执行 IO、不创建 Cobra command、不调用 SDK、不含 JSON 编码、
// 排行、磁力、详情或空列表文案。
package movie

import (
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
)

// Row 是影片列表单行的结构化投影。
type Row struct {
	Number      string
	ID          string
	Title       string
	ReleaseDate string
}

// Line 返回影片行文本 `number\tid\ttitle[\trelease_date]`（不含尾随换行）。
// 纯字符串投影，不执行 IO；空列表文案由命令负责。
func (r Row) Line() string {
	line := r.Number + "\t" + r.ID + "\t" + r.Title
	if r.ReleaseDate != "" {
		line += "\t" + r.ReleaseDate
	}
	return line
}

// Project 将影片记录投影为 Row。
// display 保留 CLI 的数值 ID 展示约定（float64 ID 截断为整数），确保输出与旧行为逐字一致。
func Project(item map[string]any) Row {
	return Row{
		Number:      display(item["number"]),
		ID:          display(item["id"]),
		Title:       display(item["title"]),
		ReleaseDate: display(item["release_date"]),
	}
}

// ProjectAll 将影片记录列表投影为 Row 列表。
func ProjectAll(items []map[string]any) []Row {
	out := make([]Row, 0, len(items))
	for _, item := range items {
		out = append(out, Project(item))
	}
	return out
}

// FilterHasMagnets 丢弃 magnets_count == 0 的行；缺失该字段的行保留。
func FilterHasMagnets(items []map[string]any) []map[string]any {
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

// display 是 CLI 影片领域的字符串展示约定：浮点 ID 截断为整数，其余委托 scalar。
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

// intValue 是 has-magnets 过滤的整数转换，保持旧 CLI 的浮点截断语义。
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
