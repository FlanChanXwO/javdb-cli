package appapi

import (
	"encoding/json"
	"strconv"
)

// MyLists GET /api/v1/lists — sort_by is required by the server.
func (c *Client) MyLists(page, limit int, sortBy string) (SearchResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if sortBy == "" {
		sortBy = "created"
	}
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/lists", map[string]string{
		"page":    strconv.Itoa(page),
		"limit":   strconv.Itoa(limit),
		"sort_by": sortBy,
	}, &data); err != nil {
		return nil, err
	}
	return SearchResult(data), nil
}

// ListInfo GET /api/v1/lists/{id} — full payload (list, is_creator, …).
func (c *Client) ListInfo(listID string) (map[string]any, error) {
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/lists/"+listID, nil, &data); err != nil {
		return nil, err
	}
	out := map[string]any{}
	for k, v := range data {
		var x any
		_ = json.Unmarshal(v, &x)
		out[k] = x
	}
	return out, nil
}

// RelatedLists GET /api/v1/lists/related?movie_id=
func (c *Client) RelatedLists(movieID string, page, limit int) (SearchResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/lists/related", map[string]string{
		"movie_id": movieID,
		"page":     strconv.Itoa(page),
		"limit":    strconv.Itoa(limit),
	}, &data); err != nil {
		return nil, err
	}
	return SearchResult(data), nil
}
