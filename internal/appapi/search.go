package appapi

import "encoding/json"

// SearchResult is a loosely typed /api/v2/search data payload.
// Keys present depend on Type (movies/codes/series/actors/makers/directors/lists).
type SearchResult map[string]json.RawMessage

// Movies unmarshals the movies array if present.
func (r SearchResult) Movies() []map[string]any {
	return decodeObjectArray(r["movies"])
}

// Named unmarshals a named dimension list (codes, series, actors, …).
func (r SearchResult) Named(key string) []map[string]any {
	return decodeObjectArray(r[key])
}

func decodeObjectArray(raw json.RawMessage) []map[string]any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

// Search calls GET /api/v2/search.
func (c *Client) Search(keyword string, opt SearchOptions) (SearchResult, error) {
	params := BuildSearchParams(keyword, opt)
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v2/search", params, &data); err != nil {
		return nil, err
	}
	return SearchResult(data), nil
}
