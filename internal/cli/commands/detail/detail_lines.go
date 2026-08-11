package detail

import (
	"fmt"
	"io"
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/magnet"
	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
)

// renderDetail 写出 graph-oriented 详情行（保持 Python parity 的既有文本）。
func renderDetail(w io.Writer, movie map[string]any) {
	fmt.Fprintf(w, "番号\t%s\n", display(movie["number"]))
	fmt.Fprintf(w, "id\t%s\n", display(movie["id"]))
	fmt.Fprintf(w, "标题\t%s\n", display(movie["title"]))
	fmt.Fprintf(w, "评分\t%s\n", display(movie["score"]))
	fmt.Fprintf(w, "日期\t%s\n", display(movie["release_date"]))
	fmt.Fprintf(w, "磁力数\t%s\n", display(movie["magnets_count"]))
	if display(movie["series_id"]) != "" || display(movie["series_name"]) != "" {
		fmt.Fprintf(w, "系列\t%s\t%s\n", display(movie["series_id"]), display(movie["series_name"]))
	}
	if display(movie["maker_id"]) != "" || display(movie["maker_name"]) != "" {
		fmt.Fprintf(w, "厂牌\t%s\t%s\n", display(movie["maker_id"]), display(movie["maker_name"]))
	}
	if display(movie["director_id"]) != "" || display(movie["director_name"]) != "" {
		fmt.Fprintf(w, "导演\t%s\t%s\n", display(movie["director_id"]), display(movie["director_name"]))
	}
	for _, a := range asSlice(movie["actors"]) {
		if m, ok := a.(map[string]any); ok {
			fmt.Fprintf(w, "演员\t%s\t%s\n", display(m["id"]), display(m["name"]))
		} else {
			fmt.Fprintf(w, "演员\t\t%v\n", a)
		}
	}
	for _, t := range asSlice(movie["tags"]) {
		if m, ok := t.(map[string]any); ok {
			fmt.Fprintf(w, "标签\t%s\t%s\n", display(m["id"]), display(m["name"]))
		} else {
			fmt.Fprintf(w, "标签\t\t%v\n", t)
		}
	}
}

// renderMagnets 用 magnet 投影写出磁力行文本；空列表输出 (无磁力链)。
func renderMagnets(w, errW io.Writer, items []map[string]any) {
	if len(items) == 0 {
		fmt.Fprintln(errW, "(无磁力链)")
		return
	}
	for _, row := range magnet.ProjectAll(items) {
		fmt.Fprintln(w, row.Line())
		fmt.Fprintln(w, row.HashLine())
	}
}

// display 保持 CLI 既有的数值 ID 截断展示约定，与 movie/magnet 投影一致。
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

func asSlice(v any) []any {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case []any:
		return t
	case []map[string]any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = t[i]
		}
		return out
	default:
		return nil
	}
}
