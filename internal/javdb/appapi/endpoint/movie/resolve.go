package movie

import (
	"context"
	"fmt"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
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
		n := strings.ToUpper(scalar.String(m["number"]))
		if n == want {
			id := scalar.String(m["id"])
			if id == "" {
				return "", fmt.Errorf("match for %s has no id", number)
			}
			return id, nil
		}
	}
	if len(movies) > 0 {
		id := scalar.String(movies[0]["id"])
		if id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("找不到番号: %s", number)
}

// ResolveNumberExact 只接受大小写不敏感的完整相等番号；零匹配与多重精确匹配
// 都显式失败，绝不回退到搜索首项。图片反搜联动必须使用本函数。
func ResolveNumberExact(movies []map[string]any, number string) (string, error) {
	want := strings.ToUpper(strings.TrimSpace(number))
	if want == "" {
		return "", fmt.Errorf("empty number")
	}
	var selected string
	for _, m := range movies {
		n := strings.ToUpper(scalar.String(m["number"]))
		if n != want {
			continue
		}
		id := scalar.String(m["id"])
		if id == "" {
			return "", fmt.Errorf("exact match for %s has no id", number)
		}
		if selected != "" {
			return "", fmt.Errorf("番号 %s 有多个精确匹配", number)
		}
		selected = id
	}
	if selected == "" {
		return "", fmt.Errorf("找不到番号: %s", number)
	}
	return selected, nil
}

// ResolveMovieID searches with zone=all and resolves number → id.
func (e *MovieEndpoint) ResolveMovieID(number string) (string, error) {
	res, err := e.search.Search(number, model.SearchOptions{Zone: "all", Page: 1})
	if err != nil {
		return "", err
	}
	return ResolveNumber(res.Movies(), number)
}

// ResolveMovieIDExact searches with zone=all and applies strict exact matching.
func (e *MovieEndpoint) ResolveMovieIDExact(ctx context.Context, number string) (string, error) {
	res, err := e.search.SearchContext(ctx, number, model.SearchOptions{Zone: "all", Page: 1, Limit: 100})
	if err != nil {
		return "", err
	}
	return ResolveNumberExact(res.Movies(), number)
}
