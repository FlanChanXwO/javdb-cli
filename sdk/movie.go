package javdb

import (
	"context"
)

// MovieDetail returns GET /api/v4/movies/{id} nested movie map.
func (c *Client) MovieDetail(ctx context.Context, movieID string) (map[string]any, error) {
	_ = ctx
	return c.api.MovieDetail(movieID)
}

// MovieMagnets returns GET /api/v1/movies/{id}/magnets (works anonymously).
func (c *Client) MovieMagnets(ctx context.Context, movieID string) ([]map[string]any, error) {
	_ = ctx
	return c.api.MovieMagnets(movieID)
}

// MovieComments returns one requested page from GET /api/v1/movies/{id}/reviews.
// It never follows later pages automatically.
func (c *Client) MovieComments(ctx context.Context, movieID string, page, limit int) ([]map[string]any, error) {
	_ = ctx
	return c.api.MovieComments(movieID, page, limit)
}

// ResolveMovieID maps a printed number to internal id (search zone=all).
func (c *Client) ResolveMovieID(ctx context.Context, number string) (string, error) {
	_ = ctx
	return c.api.ResolveMovieID(number)
}
