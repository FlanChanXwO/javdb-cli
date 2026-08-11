package movie

import (
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

// ResolveMovieID searches with zone=all and resolves number → id.
func (e *MovieEndpoint) ResolveMovieID(number string) (string, error) {
	res, err := e.search.Search(number, model.SearchOptions{Zone: "all", Page: 1})
	if err != nil {
		return "", err
	}
	return ResolveNumber(res.Movies(), number)
}
