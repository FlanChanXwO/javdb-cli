package javdb

import "context"

// WatchedMovies returns all watched (看過) movies.
func (c *Client) WatchedMovies(ctx context.Context) ([]map[string]any, error) {
	_ = ctx
	return c.api.WatchedMovies()
}

// WantMovies returns all want_watch (想看) movies.
func (c *Client) WantMovies(ctx context.Context) ([]map[string]any, error) {
	_ = ctx
	return c.api.WantMovies()
}

// Mark marks a movie watched or want_watch.
func (c *Client) Mark(ctx context.Context, movieID, status string, score int, content string) (map[string]any, error) {
	_ = ctx
	return c.api.Mark(movieID, status, score, content)
}

// Unmark removes the user's review mark.
func (c *Client) Unmark(ctx context.Context, movieID string) (bool, error) {
	_ = ctx
	return c.api.Unmark(movieID)
}

// Collected returns a user collection kind.
func (c *Client) Collected(ctx context.Context, kind string) ([]map[string]any, error) {
	_ = ctx
	return c.api.Collected(kind)
}

// RecentViewed returns recently viewed movies.
func (c *Client) RecentViewed(ctx context.Context) ([]map[string]any, error) {
	_ = ctx
	return c.api.RecentViewed()
}
