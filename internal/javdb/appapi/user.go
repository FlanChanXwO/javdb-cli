package appapi

import (
	"encoding/json"
	"fmt"
	"strconv"
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
			id := anyStr(it["id"])
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
func (c *Client) ReviewMoviesPage(status string, page int) ([]map[string]any, error) {
	if page <= 0 {
		page = 1
	}
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v2/users/review_movies", map[string]string{
		"status": status,
		"page":   strconv.Itoa(page),
	}, &data); err != nil {
		return nil, err
	}
	return decodeObjectArray(data["movies"]), nil
}

// WatchedMovies returns all watched (看過) movies.
func (c *Client) WatchedMovies() ([]map[string]any, error) {
	return AllPages(func(p int) ([]map[string]any, error) {
		return c.ReviewMoviesPage("watched", p)
	}, 100)
}

// WantMovies returns all want_watch (想看) movies.
func (c *Client) WantMovies() ([]map[string]any, error) {
	return AllPages(func(p int) ([]map[string]any, error) {
		return c.ReviewMoviesPage("want_watch", p)
	}, 100)
}

// Mark posts POST /api/v1/movies/{id}/reviews. status ∈ watched|want_watch.
func (c *Client) Mark(movieID, status string, score int, content string) (map[string]any, error) {
	if status != "watched" && status != "want_watch" {
		return nil, fmt.Errorf("status must be watched or want_watch")
	}
	var data map[string]json.RawMessage
	if err := c.PostFormJSON("/api/v1/movies/"+movieID+"/reviews", map[string]string{
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
func (c *Client) Unmark(movieID string) (bool, error) {
	detail, err := c.MovieDetail(movieID)
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
	rid := anyStr(rev["id"])
	if rid == "" {
		return false, nil
	}
	if err := c.DeleteJSON("/api/v1/movies/"+movieID+"/reviews/"+rid, nil, nil); err != nil {
		return false, err
	}
	return true, nil
}

// CollectedPage is one page of a user collection kind.
func (c *Client) CollectedPage(kind string, page int) ([]map[string]any, error) {
	spec, ok := CollectionSpecs[kind]
	if !ok {
		return nil, fmt.Errorf("collection kind must be one of actors|series|codes|makers|directors")
	}
	if page <= 0 {
		page = 1
	}
	var data map[string]json.RawMessage
	if err := c.GetJSON(spec.Path, map[string]string{"page": strconv.Itoa(page)}, &data); err != nil {
		return nil, err
	}
	return decodeObjectArray(data[spec.Key]), nil
}

// Collected aggregates all pages of a collection kind.
func (c *Client) Collected(kind string) ([]map[string]any, error) {
	if _, ok := CollectionSpecs[kind]; !ok {
		return nil, fmt.Errorf("collection kind must be one of actors|series|codes|makers|directors")
	}
	return AllPages(func(p int) ([]map[string]any, error) {
		return c.CollectedPage(kind, p)
	}, 100)
}

// RecentViewed returns GET /api/v1/users/recent_viewed movies (unpaged).
func (c *Client) RecentViewed() ([]map[string]any, error) {
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/users/recent_viewed", nil, &data); err != nil {
		return nil, err
	}
	return decodeObjectArray(data["movies"]), nil
}
