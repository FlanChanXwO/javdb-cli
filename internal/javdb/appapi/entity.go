package appapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// EntityMoviesOptions for filmography listing.
type EntityMoviesOptions struct {
	Zone  string
	Page  int
	Limit int
	Sort  string
	Order string
	Main  []string
	Tags  []string // resolved tag ids
}

// EntityMovies lists movies for an entity via GET /api/v1/movies/tags.
func (c *Client) EntityMovies(kind, entityID string, opt EntityMoviesOptions) (SearchResult, error) {
	if opt.Zone == "" {
		opt.Zone = "censored"
	}
	if opt.Page <= 0 {
		opt.Page = 1
	}
	if opt.Limit <= 0 {
		opt.Limit = 20
	}
	if opt.Sort == "" {
		opt.Sort = "release"
	}
	if opt.Order == "" {
		opt.Order = "desc"
	}
	mask, err := BuildEntityFilter(kind, entityID, opt.Zone, opt.Main)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"filter_by":      mask,
		"filter_by_tags": strings.Join(opt.Tags, ","),
		"sort_by":        opt.Sort,
		"order_by":       opt.Order,
		"page":           strconv.Itoa(opt.Page),
		"limit":          strconv.Itoa(opt.Limit),
	}
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/movies/tags", params, &data); err != nil {
		return nil, err
	}
	return SearchResult(data), nil
}

// EntityDetail fetches GET /api/v1/{kind}s/{id} (or lists/{id} for list).
func (c *Client) EntityDetail(kind, id string) (map[string]any, error) {
	var path string
	switch kind {
	case "actor":
		path = "/api/v1/actors/" + id
	case "series":
		path = "/api/v1/series/" + id
	case "maker":
		path = "/api/v1/makers/" + id
	case "director":
		path = "/api/v1/directors/" + id
	case "code":
		path = "/api/v1/codes/" + id
	case "list":
		path = "/api/v1/lists/" + id
	default:
		return nil, fmt.Errorf("unknown entity kind: %s", kind)
	}
	var data map[string]json.RawMessage
	if err := c.GetJSON(path, nil, &data); err != nil {
		return nil, err
	}
	// nest under kind or "list"
	key := kind
	if kind == "list" {
		key = "list"
	}
	if raw, ok := data[key]; ok && len(raw) > 0 && string(raw) != "null" {
		var nested map[string]any
		if err := json.Unmarshal(raw, &nested); err == nil {
			return nested, nil
		}
	}
	// flat fallback
	out := map[string]any{}
	for k, v := range data {
		var x any
		_ = json.Unmarshal(v, &x)
		out[k] = x
	}
	return out, nil
}

// ResolveEntity resolves id or name to entity id.
func (c *Client) ResolveEntity(kind, ref, zone string) (string, error) {
	if _, ok := EntityLetters[kind]; !ok {
		return "", fmt.Errorf("unknown entity kind: %s", kind)
	}
	if zone == "" {
		zone = "censored"
	}
	// 1) treat as id
	if meta, err := c.EntityDetail(kind, ref); err == nil {
		if id := anyStr(meta["id"]); id != "" {
			return id, nil
		}
		// some payloads only nest without id field on success of path
		return ref, nil
	}
	// 2) search by name
	searchZone := zone
	if zone == "all" {
		searchZone = "all"
	}
	res, err := c.Search(ref, SearchOptions{Type: kind, Zone: searchZone, Page: 1})
	if err != nil {
		return "", err
	}
	key := SearchTypeListKey(kind)
	items := res.Named(key)
	refCF := strings.ToLower(strings.TrimSpace(ref))
	for _, it := range items {
		name := strings.ToLower(anyStr(it["name"]))
		if name == "" {
			name = strings.ToLower(anyStr(it["name_zht"]))
		}
		if name == refCF || strings.ToLower(anyStr(it["id"])) == refCF {
			return anyStr(it["id"]), nil
		}
	}
	if len(items) > 0 {
		return anyStr(items[0]["id"]), nil
	}
	return "", &Error{Action: "NotFound", Message: fmt.Sprintf("no %s matching %q", kind, ref)}
}

// SearchTypeListKey maps entity kind to search response list key.
func SearchTypeListKey(kind string) string {
	switch kind {
	case "actor":
		return "actors"
	case "series":
		return "series"
	case "maker":
		return "makers"
	case "director":
		return "directors"
	case "code":
		return "codes"
	case "list":
		return "lists"
	default:
		return kind + "s"
	}
}

// AllEntityMovies pages until empty (cap maxPages).
func (c *Client) AllEntityMovies(kind, entityID string, opt EntityMoviesOptions, maxPages int) ([]map[string]any, error) {
	if maxPages <= 0 {
		maxPages = 50
	}
	if opt.Page <= 0 {
		opt.Page = 1
	}
	var all []map[string]any
	seen := map[string]bool{}
	for i := 0; i < maxPages; i++ {
		opt.Page = i + 1
		res, err := c.EntityMovies(kind, entityID, opt)
		if err != nil {
			return all, err
		}
		page := res.Movies()
		if len(page) == 0 {
			break
		}
		for _, m := range page {
			id := anyStr(m["id"])
			if id != "" {
				if seen[id] {
					continue
				}
				seen[id] = true
			}
			all = append(all, m)
		}
	}
	return all, nil
}
