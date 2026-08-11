package result

import (
	"fmt"
)

// MagnetRow 是磁力单行的结构化投影。
type MagnetRow struct {
	Name      string
	Size      string
	Flags     []string
	CreatedAt string
	Hash      string
}

// Line 返回磁力行文本 `name\tsize\tflags\tdate`（无标记时为 `-`，不含尾随换行）。
func (r MagnetRow) Line() string {
	flagS := "-"
	if len(r.Flags) > 0 {
		flagS = joinComma(r.Flags)
	}
	return r.Name + "\t" + r.Size + "\t" + flagS + "\t" + r.CreatedAt
}

// HashLine 返回 `  magnet:?xt=urn:btih:{hash}` 行。
func (r MagnetRow) HashLine() string {
	return "  magnet:?xt=urn:btih:" + r.Hash
}

// ProjectMagnet 将磁力记录投影为 MagnetRow，保持名称降级、日期截断与 hash 语义。
func ProjectMagnet(item map[string]any) MagnetRow {
	name := display(item["name"])
	if name == "" {
		name = display(item["title"])
	}
	date := display(item["created_at"])
	if len(date) > 10 {
		date = date[:10]
	}
	return MagnetRow{
		Name:      name,
		Size:      formatSize(item["size"]),
		Flags:     flags(item),
		CreatedAt: date,
		Hash:      display(item["hash"]),
	}
}

// ProjectMagnets 将磁力记录列表投影为 MagnetRow 列表。
func ProjectMagnets(items []map[string]any) []MagnetRow {
	out := make([]MagnetRow, 0, len(items))
	for _, item := range items {
		out = append(out, ProjectMagnet(item))
	}
	return out
}

// flags 返回按顺序出现的 truthy 标记（cnsub、hd）。
func flags(item map[string]any) []string {
	var out []string
	if truthy(item["cnsub"]) {
		out = append(out, "cnsub")
	}
	if truthy(item["hd"]) {
		out = append(out, "hd")
	}
	return out
}

// formatSize 格式化磁力大小（MiB 整数），保持 GB/MB 与降级语义。
func formatSize(size any) string {
	n := intValue(size)
	if n <= 0 && size != nil {
		if f, ok := size.(float64); ok {
			n = int(f)
		}
	}
	if n >= 1024 {
		return fmt.Sprintf("%.1fGB", float64(n)/1024)
	}
	if n > 0 {
		return fmt.Sprintf("%dMB", n)
	}
	return display(size)
}

func joinComma(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for i := 1; i < len(ss); i++ {
		out += "," + ss[i]
	}
	return out
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case int:
		return t != 0
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}
