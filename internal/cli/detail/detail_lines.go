package detail

import (
	"fmt"
	"io"
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	"github.com/FlanChanXwO/javdb-cli/internal/display"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
)

// renderDetail 写出 graph-oriented 详情行（保持 Python parity 的既有文本）。
func renderDetail(w io.Writer, movie map[string]any) {
	fmt.Fprintf(w, i18n.T("cli.s"), displayValue(movie["number"]))
	fmt.Fprintf(w, i18n.T("cli.id_s"), displayValue(movie["id"]))
	fmt.Fprintf(w, i18n.T("cli.s_2"), displayValue(movie["title"]))
	fmt.Fprintf(w, i18n.T("cli.s_3"), displayValue(movie["score"]))
	fmt.Fprintf(w, i18n.T("cli.s_4"), displayValue(movie["release_date"]))
	if minutes, ok := display.MovieDurationMinutes(movie); ok {
		fmt.Fprintf(w, i18n.T("cli.d"), minutes)
	}
	fmt.Fprintf(w, i18n.T("cli.s_5"), displayValue(movie["magnets_count"]))
	if displayValue(movie["series_id"]) != "" || displayValue(movie["series_name"]) != "" {
		fmt.Fprintf(w, i18n.T("cli.s_s_3"), displayValue(movie["series_id"]), displayValue(movie["series_name"]))
	}
	if displayValue(movie["maker_id"]) != "" || displayValue(movie["maker_name"]) != "" {
		fmt.Fprintf(w, i18n.T("cli.s_s_4"), displayValue(movie["maker_id"]), displayValue(movie["maker_name"]))
	}
	if displayValue(movie["director_id"]) != "" || displayValue(movie["director_name"]) != "" {
		fmt.Fprintf(w, i18n.T("cli.s_s_5"), displayValue(movie["director_id"]), displayValue(movie["director_name"]))
	}
	for _, a := range asSlice(movie["actors"]) {
		if m, ok := a.(map[string]any); ok {
			fmt.Fprintf(w, i18n.T("cli.s_s_6"), displayValue(m["id"]), displayValue(m["name"]))
		} else {
			fmt.Fprintf(w, i18n.T("cli.v"), a)
		}
	}
	for _, t := range asSlice(movie["tags"]) {
		if m, ok := t.(map[string]any); ok {
			fmt.Fprintf(w, i18n.T("cli.s_s_7"), displayValue(m["id"]), displayValue(m["name"]))
		} else {
			fmt.Fprintf(w, i18n.T("cli.v_2"), t)
		}
	}
}

// renderMagnets 用 magnet 投影写出磁力行文本；空列表输出 (无磁力链)。
func renderMagnets(w, errW io.Writer, items []map[string]any) {
	if len(items) == 0 {
		fmt.Fprintln(errW, i18n.T("cli.message"))
		return
	}
	for _, row := range display.ProjectMagnets(items) {
		fmt.Fprintln(w, row.Line())
		fmt.Fprintln(w, row.HashLine())
	}
}

// display 保持 CLI 既有的数值 ID 截断展示约定，与 movie/magnet 投影一致。
func displayValue(v any) string {
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
