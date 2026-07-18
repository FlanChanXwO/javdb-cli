package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

// FilterHasMagnets drops rows with magnets_count == 0; keeps missing field.
func FilterHasMagnets(movies []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(movies))
	for _, m := range movies {
		if v, ok := m["magnets_count"]; ok {
			n := anyToInt(v)
			if n == 0 {
				continue
			}
		}
		out = append(out, m)
	}
	return out
}

func anyToInt(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(t)
		return n
	default:
		return 0
	}
}

func anyString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		// ids sometimes decode as numbers — uncommon for movie ids
		return strconv.FormatInt(int64(t), 10)
	default:
		return fmt.Sprint(t)
	}
}

// PrintMovies writes number\tid\ttitle[\tdate] lines.
func PrintMovies(w io.Writer, errW io.Writer, movies []map[string]any) {
	if len(movies) == 0 {
		fmt.Fprintln(errW, "(空列表)")
		return
	}
	for _, m := range movies {
		line := fmt.Sprintf("%s\t%s\t%s", anyString(m["number"]), anyString(m["id"]), anyString(m["title"]))
		if d := anyString(m["release_date"]); d != "" {
			line += "\t" + d
		}
		fmt.Fprintln(w, line)
	}
}

// PrintNamed writes id\tname[\tcount] for codes/series/actors/….
func PrintNamed(w io.Writer, errW io.Writer, items []map[string]any) {
	if len(items) == 0 {
		fmt.Fprintln(errW, "(空列表)")
		return
	}
	for _, it := range items {
		name := anyString(it["name_zht"])
		if name == "" {
			name = anyString(it["name"])
		}
		count := it["videos_count"]
		if count == nil {
			count = it["movies_count"]
		}
		if count == nil {
			fmt.Fprintf(w, "%s\t%s\n", anyString(it["id"]), name)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%v\n", anyString(it["id"]), name, count)
		}
	}
}

// EmitJSON writes compact JSON.
func EmitJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// SearchTypeKey maps --type to response list key.
func SearchTypeKey(type_ string) string {
	switch type_ {
	case "movie", "":
		return "movies"
	case "code":
		return "codes"
	case "series":
		return "series"
	case "actor":
		return "actors"
	case "maker":
		return "makers"
	case "director":
		return "directors"
	case "list":
		return "lists"
	default:
		return "movies"
	}
}
