package javdb

import (
	"context"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi"
)

// EntityMoviesOptions re-export.
type EntityMoviesOptions = appapi.EntityMoviesOptions

// EntityMovies lists filmography for an entity kind.
func (c *Client) EntityMovies(ctx context.Context, kind, entityID string, opt EntityMoviesOptions) (SearchResult, error) {
	_ = ctx
	return c.api.EntityMovies(kind, entityID, opt)
}

// ResolveEntity resolves name/id to entity id.
func (c *Client) ResolveEntity(ctx context.Context, kind, ref, zone string) (string, error) {
	_ = ctx
	return c.api.ResolveEntity(kind, ref, zone)
}

// EntityDetail fetches entity meta.
func (c *Client) EntityDetail(ctx context.Context, kind, id string) (map[string]any, error) {
	_ = ctx
	return c.api.EntityDetail(kind, id)
}

// AllEntityMovies aggregates filmography pages.
func (c *Client) AllEntityMovies(ctx context.Context, kind, entityID string, opt EntityMoviesOptions, maxPages int) ([]map[string]any, error) {
	_ = ctx
	return c.api.AllEntityMovies(kind, entityID, opt, maxPages)
}
