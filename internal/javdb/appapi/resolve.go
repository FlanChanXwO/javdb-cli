package appapi

import (
	"fmt"
	"strings"
)

// ResolveNumber finds the internal movie id for a printed number (e.g. SSIS-589).
// Prefers an exact case-insensitive number match; else first search hit.
// Uses zone "all" (omit movie_type) so uncensored/western/fc2 still resolve.
func ResolveNumber(movies []map[string]any, number string) (string, error) {
	want := strings.ToUpper(strings.TrimSpace(number))
	if want == "" {
		return "", fmt.Errorf("empty number")
	}
	for _, m := range movies {
		n := strings.ToUpper(anyStr(m["number"]))
		if n == want {
			id := anyStr(m["id"])
			if id == "" {
				return "", fmt.Errorf("match for %s has no id", number)
			}
			return id, nil
		}
	}
	if len(movies) > 0 {
		id := anyStr(movies[0]["id"])
		if id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("找不到番号: %s", number)
}

func anyStr(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

// ResolveMovieID searches with zone=all and resolves number → id.
func (c *Client) ResolveMovieID(number string) (string, error) {
	res, err := c.Search(number, SearchOptions{Zone: "all", Page: 1})
	if err != nil {
		return "", err
	}
	return ResolveNumber(res.Movies(), number)
}
