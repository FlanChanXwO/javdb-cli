package javdb

import "context"

// MyLists returns the current user's playlists (auth).
func (c *Client) MyLists(ctx context.Context, page, limit int, sortBy string) (SearchResult, error) {
	_ = ctx
	return c.api.MyLists(page, limit, sortBy)
}

// ListInfo returns list meta payload.
func (c *Client) ListInfo(ctx context.Context, listID string) (map[string]any, error) {
	_ = ctx
	return c.api.ListInfo(listID)
}

// RelatedLists returns public lists related to a movie.
func (c *Client) RelatedLists(ctx context.Context, movieID string, page, limit int) (SearchResult, error) {
	_ = ctx
	return c.api.RelatedLists(movieID, page, limit)
}
