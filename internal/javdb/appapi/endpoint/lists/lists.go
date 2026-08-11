package lists

import (
	"encoding/json"
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/client"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
)

// ListsEndpoint 提供用户列表与相关列表 capability。
type ListsEndpoint struct {
	c *client.Client
}

// NewLists 用共享 transport 构造 lists capability。
func NewLists(c *client.Client) *ListsEndpoint {
	return &ListsEndpoint{c: c}
}

// MyLists GET /api/v1/lists — sort_by is required by the server.
func (e *ListsEndpoint) MyLists(page, limit int, sortBy string) (model.SearchResult, error) {
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
	if err := e.c.GetJSON("/api/v1/lists", map[string]string{
		"page":    strconv.Itoa(page),
		"limit":   strconv.Itoa(limit),
		"sort_by": sortBy,
	}, &data); err != nil {
		return nil, err
	}
	return model.SearchResult(data), nil
}

// ListInfo GET /api/v1/lists/{id} — full payload (list, is_creator, …).
func (e *ListsEndpoint) ListInfo(listID string) (map[string]any, error) {
	var data map[string]json.RawMessage
	if err := e.c.GetJSON("/api/v1/lists/"+listID, nil, &data); err != nil {
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
func (e *ListsEndpoint) RelatedLists(movieID string, page, limit int) (model.SearchResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	var data map[string]json.RawMessage
	if err := e.c.GetJSON("/api/v1/lists/related", map[string]string{
		"movie_id": movieID,
		"page":     strconv.Itoa(page),
		"limit":    strconv.Itoa(limit),
	}, &data); err != nil {
		return nil, err
	}
	return model.SearchResult(data), nil
}
