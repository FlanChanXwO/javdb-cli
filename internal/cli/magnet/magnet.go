// Package magnet 提供磁力记录的纯投影与大小格式化。
//
// detail --magnets 与 magnets 命令共同使用本投影；包只返回结构化 Row，命令负责写出既有文本。
// 不执行 IO、不创建 Cobra command、不调用 SDK、不含 JSON 编码或空列表文案。
package magnet

import (
	"fmt"
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
)

// Row 是磁力单行的结构化投影。
type Row struct {
	Name      string
	Size      string
	Flags     []string
	CreatedAt string
	Hash      string
}

// Line 返回磁力行文本 `name\tsize\tflags\tdate`（无标记时为 `-`，不含尾随换行）。
// HashLine 返回 `  magnet:?xt=urn:btih:{hash}`。均为纯字符串投影，不执行 IO；
// 空列表文案由命令负责。
func (r Row) Line() string {
	flagS := "-"
	if len(r.Flags) > 0 {
		flagS = joinComma(r.Flags)
	}
	return r.Name + "\t" + r.Size + "\t" + flagS + "\t" + r.CreatedAt
}

func (r Row) HashLine() string {
	return "  magnet:?xt=urn:btih:" + r.Hash
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

// Project 将磁力记录投影为 Row，保持 CLI 既有的名称降级、日期截断和 hash 语义。
func Project(item map[string]any) Row {
	name := display(item["name"])
	if name == "" {
		name = display(item["title"])
	}
	date := display(item["created_at"])
	if len(date) > 10 {
		date = date[:10]
	}
	return Row{
		Name:      name,
		Size:      FormatSize(item["size"]),
		Flags:     Flags(item),
		CreatedAt: date,
		Hash:      display(item["hash"]),
	}
}

// ProjectAll 将磁力记录列表投影为 Row 列表。
func ProjectAll(items []map[string]any) []Row {
	out := make([]Row, 0, len(items))
	for _, item := range items {
		out = append(out, Project(item))
	}
	return out
}

// Flags 返回按顺序出现的 truthy 标记（cnsub、hd）。
func Flags(item map[string]any) []string {
	var flags []string
	if truthy(item["cnsub"]) {
		flags = append(flags, "cnsub")
	}
	if truthy(item["hd"]) {
		flags = append(flags, "hd")
	}
	return flags
}

// FormatSize 格式化磁力大小（MiB 整数），保持 CLI 既有的 GB/MB 与降级语义。
func FormatSize(size any) string {
	n := intValue(size)
	if n <= 0 && size != nil {
		// 尝试 float
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

// display 是 CLI 磁力领域的字符串展示约定：浮点值截断为整数，其余委托 scalar。
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
