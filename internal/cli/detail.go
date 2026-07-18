package cli

import (
	"fmt"
	"io"
)

// PrintDetail writes graph-oriented detail lines (Python parity).
func PrintDetail(w io.Writer, movie map[string]any) {
	fmt.Fprintf(w, "番号\t%s\n", anyString(movie["number"]))
	fmt.Fprintf(w, "id\t%s\n", anyString(movie["id"]))
	fmt.Fprintf(w, "标题\t%s\n", anyString(movie["title"]))
	fmt.Fprintf(w, "评分\t%s\n", anyString(movie["score"]))
	fmt.Fprintf(w, "日期\t%s\n", anyString(movie["release_date"]))
	fmt.Fprintf(w, "磁力数\t%s\n", anyString(movie["magnets_count"]))
	if anyString(movie["series_id"]) != "" || anyString(movie["series_name"]) != "" {
		fmt.Fprintf(w, "系列\t%s\t%s\n", anyString(movie["series_id"]), anyString(movie["series_name"]))
	}
	if anyString(movie["maker_id"]) != "" || anyString(movie["maker_name"]) != "" {
		fmt.Fprintf(w, "厂牌\t%s\t%s\n", anyString(movie["maker_id"]), anyString(movie["maker_name"]))
	}
	if anyString(movie["director_id"]) != "" || anyString(movie["director_name"]) != "" {
		fmt.Fprintf(w, "导演\t%s\t%s\n", anyString(movie["director_id"]), anyString(movie["director_name"]))
	}
	for _, a := range asSlice(movie["actors"]) {
		if m, ok := a.(map[string]any); ok {
			fmt.Fprintf(w, "演员\t%s\t%s\n", anyString(m["id"]), anyString(m["name"]))
		} else {
			fmt.Fprintf(w, "演员\t\t%v\n", a)
		}
	}
	for _, t := range asSlice(movie["tags"]) {
		if m, ok := t.(map[string]any); ok {
			fmt.Fprintf(w, "标签\t%s\t%s\n", anyString(m["id"]), anyString(m["name"]))
		} else {
			fmt.Fprintf(w, "标签\t\t%v\n", t)
		}
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

// PrintMovieMagnets prints magnet rows (name/size/flags/hash). Basic form for detail --magnets.
func PrintMovieMagnets(w io.Writer, errW io.Writer, items []map[string]any) {
	if len(items) == 0 {
		fmt.Fprintln(errW, "(无磁力链)")
		return
	}
	for _, m := range items {
		name := anyString(m["name"])
		if name == "" {
			name = anyString(m["title"])
		}
		size := FmtSize(m["size"])
		var flags []string
		if truthy(m["cnsub"]) {
			flags = append(flags, "cnsub")
		}
		if truthy(m["hd"]) {
			flags = append(flags, "hd")
		}
		flagS := "-"
		if len(flags) > 0 {
			flagS = joinComma(flags)
		}
		date := anyString(m["created_at"])
		if len(date) > 10 {
			date = date[:10]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, size, flagS, date)
		fmt.Fprintf(w, "  magnet:?xt=urn:btih:%s\n", anyString(m["hash"]))
	}
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
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

// FmtSize formats magnet size (MiB integer from API).
func FmtSize(size any) string {
	n := anyToInt(size)
	if n <= 0 && size != nil {
		// try float
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
	return anyString(size)
}
