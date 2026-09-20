package movie

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
)

// ResolveNumber finds the internal movie id for a printed number.
// It prefers an exact case-insensitive match, then accepts one unambiguous
// alphanumeric formatting-equivalent candidate; it never picks a first hit.
func ResolveNumber(movies []map[string]any, number string) (string, error) {
	normalized := strings.TrimSpace(number)
	if normalized == "" {
		return "", fmt.Errorf("empty number")
	}
	want := strings.ToUpper(normalized)
	var selected string
	for _, m := range movies {
		n := strings.ToUpper(scalar.String(m["number"]))
		if n == want {
			id := scalar.String(m["id"])
			if id == "" {
				return "", fmt.Errorf("match for %s has no id", number)
			}
			if selected != "" && selected != id {
				return "", fmt.Errorf("番号 %s 有多个精确匹配", number)
			}
			selected = id
		}
	}
	if selected != "" {
		return selected, nil
	}

	wantKey := movieNumberKey(normalized)
	for _, m := range movies {
		if movieNumberKey(scalar.String(m["number"])) != wantKey {
			continue
		}
		id := scalar.String(m["id"])
		if id == "" {
			return "", fmt.Errorf("match for %s has no id", number)
		}
		if selected != "" && selected != id {
			return "", fmt.Errorf("番号 %s 有多个格式等价匹配", number)
		}
		selected = id
	}
	if selected != "" {
		return selected, nil
	}
	return "", fmt.Errorf("找不到番号: %s", number)
}

// movieNumberKey returns a case-insensitive key that ignores formatting-only
// separators while preserving every letter and digit in the movie number.
func movieNumberKey(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToUpper(r)
		}
		return -1
	}, strings.TrimSpace(s))
}

// ResolveNumberExact 只接受大小写不敏感的完整相等番号；零匹配与多个不同 ID 的精确匹配
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
		if selected != "" && selected != id {
			return "", fmt.Errorf("番号 %s 有多个精确匹配", number)
		}
		selected = id
	}
	if selected == "" {
		return "", fmt.Errorf("找不到番号: %s", number)
	}
	return selected, nil
}

// ResolveMovieID keeps the legacy signature while using safe tolerant resolution.
func (e *MovieEndpoint) ResolveMovieID(number string) (string, error) {
	normalized := strings.TrimSpace(number)
	if normalized == "" {
		return "", fmt.Errorf("empty number")
	}
	res, err := e.search.SearchContext(context.Background(), normalized, model.SearchOptions{Zone: "all", Page: 1, Limit: 100})
	if err != nil {
		return "", err
	}
	return ResolveNumber(res.Movies(), normalized)
}

// ResolveMovieIDExact searches with zone=all and applies strict exact matching.
func (e *MovieEndpoint) ResolveMovieIDExact(ctx context.Context, number string) (string, error) {
	normalized := strings.TrimSpace(number)
	if normalized == "" {
		return "", fmt.Errorf("empty number")
	}
	res, err := e.search.SearchContext(ctx, normalized, model.SearchOptions{Zone: "all", Page: 1, Limit: 100})
	if err != nil {
		return "", err
	}
	return ResolveNumberExact(res.Movies(), normalized)
}
