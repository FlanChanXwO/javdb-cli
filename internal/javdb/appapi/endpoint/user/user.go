package user

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/client"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/codec"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/endpoint/movie"
)

// CollectionSpecs maps collection kind → (path, response list key).
// collected_lists is intentionally omitted (server 500).
var CollectionSpecs = map[string]struct {
	Path string
	Key  string
}{
	"actors":    {"/api/v1/users/collected_actors", "actors"},
	"series":    {"/api/v1/users/collected_series", "series"},
	"codes":     {"/api/v1/users/collected_codes", "codes"},
	"makers":    {"/api/v1/users/collected_makers", "makers"},
	"directors": {"/api/v1/users/collected_directors", "directors"},
}

// UserEndpoint 提供用户 review、标记与收藏 capability。
type UserEndpoint struct {
	c     *client.Client
	movie *movie.MovieEndpoint
}

// NewUser 用共享 transport 与 movie capability 构造 user capability。
func NewUser(c *client.Client, movie *movie.MovieEndpoint) *UserEndpoint {
	return &UserEndpoint{c: c, movie: movie}
}

// AllPages aggregates pages until empty, de-duplicating by id.
func AllPages(fetch func(page int) ([]map[string]any, error), maxPages int) ([]map[string]any, error) {
	if maxPages <= 0 {
		maxPages = 100
	}
	var out []map[string]any
	seen := map[string]bool{}
	for page := 1; page <= maxPages; page++ {
		items, err := fetch(page)
		if err != nil {
			return out, err
		}
		if len(items) == 0 {
			break
		}
		for _, it := range items {
			id := scalar.String(it["id"])
			if id != "" {
				if seen[id] {
					continue
				}
				seen[id] = true
			}
			out = append(out, it)
		}
	}
	return out, nil
}

// ReviewMoviesPage is one page of GET /api/v2/users/review_movies.
func (e *UserEndpoint) ReviewMoviesPage(status string, page int) ([]map[string]any, error) {
	if page <= 0 {
		page = 1
	}
	var data map[string]json.RawMessage
	if err := e.c.GetJSON("/api/v2/users/review_movies", map[string]string{
		"status": status,
		"page":   strconv.Itoa(page),
	}, &data); err != nil {
		return nil, err
	}
	return codec.ObjectArray(data["movies"]), nil
}

// WatchedMovies returns all watched (看過) movies.
func (e *UserEndpoint) WatchedMovies() ([]map[string]any, error) {
	return AllPages(func(p int) ([]map[string]any, error) {
		return e.ReviewMoviesPage("watched", p)
	}, 100)
}

// WantMovies returns all want_watch (想看) movies.
func (e *UserEndpoint) WantMovies() ([]map[string]any, error) {
	return AllPages(func(p int) ([]map[string]any, error) {
		return e.ReviewMoviesPage("want_watch", p)
	}, 100)
}

// Mark posts POST /api/v1/movies/{id}/reviews. status ∈ watched|want_watch.
func (e *UserEndpoint) Mark(movieID, status string, score int, content string) (map[string]any, error) {
	if status != "watched" && status != "want_watch" {
		return nil, fmt.Errorf("status must be watched or want_watch")
	}
	var data map[string]json.RawMessage
	if err := e.c.PostFormJSON("/api/v1/movies/"+movieID+"/reviews", map[string]string{
		"status":  status,
		"score":   strconv.Itoa(score),
		"content": content,
	}, &data); err != nil {
		return nil, err
	}
	if raw, ok := data["review"]; ok && len(raw) > 0 && string(raw) != "null" {
		var rev map[string]any
		if err := json.Unmarshal(raw, &rev); err == nil {
			return rev, nil
		}
	}
	out := map[string]any{}
	for k, v := range data {
		var x any
		_ = json.Unmarshal(v, &x)
		out[k] = x
	}
	return out, nil
}

// Unmark deletes the user's review for a movie. Returns false if none.
func (e *UserEndpoint) Unmark(movieID string) (bool, error) {
	detail, err := e.movie.MovieDetail(movieID)
	if err != nil {
		return false, err
	}
	rev, _ := detail["review"].(map[string]any)
	if rev == nil {
		// review may be nested raw
		if raw, ok := detail["review"]; ok && raw != nil {
			b, _ := json.Marshal(raw)
			_ = json.Unmarshal(b, &rev)
		}
	}
	if rev == nil {
		return false, nil
	}
	rid := scalar.String(rev["id"])
	if rid == "" {
		return false, nil
	}
	if err := e.c.DeleteJSON("/api/v1/movies/"+movieID+"/reviews/"+rid, nil, nil); err != nil {
		return false, err
	}
	return true, nil
}

// CollectedPage is one page of a user collection kind.
func (e *UserEndpoint) CollectedPage(kind string, page int) ([]map[string]any, error) {
	spec, ok := CollectionSpecs[kind]
	if !ok {
		return nil, fmt.Errorf("collection kind must be one of actors|series|codes|makers|directors")
	}
	if page <= 0 {
		page = 1
	}
	var data map[string]json.RawMessage
	if err := e.c.GetJSON(spec.Path, map[string]string{"page": strconv.Itoa(page)}, &data); err != nil {
		return nil, err
	}
	return codec.ObjectArray(data[spec.Key]), nil
}

// Collected aggregates all pages of a collection kind.
func (e *UserEndpoint) Collected(kind string) ([]map[string]any, error) {
	if _, ok := CollectionSpecs[kind]; !ok {
		return nil, fmt.Errorf("collection kind must be one of actors|series|codes|makers|directors")
	}
	return AllPages(func(p int) ([]map[string]any, error) {
		return e.CollectedPage(kind, p)
	}, 100)
}

// RecentViewed returns GET /api/v1/users/recent_viewed movies (unpaged).
func (e *UserEndpoint) RecentViewed() ([]map[string]any, error) {
	var data map[string]json.RawMessage
	if err := e.c.GetJSON("/api/v1/users/recent_viewed", nil, &data); err != nil {
		return nil, err
	}
	return codec.ObjectArray(data["movies"]), nil
}
