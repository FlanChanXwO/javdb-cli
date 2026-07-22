package javdb

import (
	"context"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/tags"
)

// BrowseOptions re-export.
type BrowseOptions = appapi.BrowseOptions

// RefreshTagTaxonomy downloads EN+ZH taxonomies and saves tags-{zone}.json.
func (c *Client) RefreshTagTaxonomy(ctx context.Context, zone string) (*tags.Doc, error) {
	_ = ctx
	return c.api.RefreshTagTaxonomy(zone)
}

// LoadOrRefreshTaxonomy loads disk taxonomy or refreshes.
func (c *Client) LoadOrRefreshTaxonomy(ctx context.Context, zone string, force bool) (*tags.Doc, string, error) {
	_ = ctx
	return c.api.LoadOrRefreshTaxonomy(zone, force)
}

// ResolveTags maps free-form tag refs to ids.
func (c *Client) ResolveTags(ctx context.Context, refs []string, zone string) ([]string, error) {
	_ = ctx
	return c.api.ResolveTags(refs, zone)
}

// Browse lists movies by category filters.
func (c *Client) Browse(ctx context.Context, opt BrowseOptions) (SearchResult, error) {
	_ = ctx
	return c.api.Browse(opt)
}
