package search

import (
	"context"
	"encoding/json"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/client"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
)

// SearchEndpoint 提供 /api/v2/search capability。
type SearchEndpoint struct {
	c *client.Client
}

// NewSearch 用共享 transport 构造 search capability。
func NewSearch(c *client.Client) *SearchEndpoint {
	return &SearchEndpoint{c: c}
}

// Search calls GET /api/v2/search.
func (e *SearchEndpoint) Search(keyword string, opt SearchOptions) (model.SearchResult, error) {
	return e.SearchContext(context.Background(), keyword, opt)
}

// SearchContext calls GET /api/v2/search with an explicit context.
func (e *SearchEndpoint) SearchContext(ctx context.Context, keyword string, opt SearchOptions) (model.SearchResult, error) {
	params := BuildSearchParams(keyword, opt)
	var data map[string]json.RawMessage
	if err := e.c.GetJSONContext(ctx, "/api/v2/search", params, &data); err != nil {
		return nil, err
	}
	return model.SearchResult(data), nil
}
