package javdb

import (
	"context"

	"github.com/FlanChanXwO/javdb-cli/internal/appapi"
)

// SearchOptions is re-exported for SDK callers.
type SearchOptions = appapi.SearchOptions

// SearchResult is re-exported.
type SearchResult = appapi.SearchResult

// Search runs GET /api/v2/search.
func (c *Client) Search(ctx context.Context, keyword string, opt SearchOptions) (SearchResult, error) {
	_ = ctx
	return c.api.Search(keyword, opt)
}
