package session

import (
	"context"

	"github.com/FlanChanXwO/javdb-cli/internal/storage/tags"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

func call[T any](c *Client, ctx context.Context, required bool, fn func() (T, error)) (value T, err error) {
	if required {
		if err = c.requireAccount(); err != nil {
			return value, err
		}
	}
	err = c.execute(ctx, func() error { var callErr error; value, callErr = fn(); return callErr })
	return value, err
}

// MovieAssets executes the SDK operation under the CLI authentication policy.
func (c *Client) MovieAssets(ctx context.Context, movieID string) ([]javdb.MovieAsset, error) {
	return call(c, ctx, false, func() ([]javdb.MovieAsset, error) { return c.Client.MovieAssets(ctx, movieID) })
}

// MyLists executes the SDK operation under the CLI authentication policy.
func (c *Client) MyLists(ctx context.Context, page, limit int, sortBy string) (javdb.SearchResult, error) {
	return call(c, ctx, true, func() (javdb.SearchResult, error) { return c.Client.MyLists(ctx, page, limit, sortBy) })
}

// ListInfo executes the SDK operation under the CLI authentication policy.
func (c *Client) ListInfo(ctx context.Context, listID string) (map[string]any, error) {
	return call(c, ctx, false, func() (map[string]any, error) { return c.Client.ListInfo(ctx, listID) })
}

// RelatedLists executes the SDK operation under the CLI authentication policy.
func (c *Client) RelatedLists(ctx context.Context, movieID string, page, limit int) (javdb.SearchResult, error) {
	return call(c, ctx, false, func() (javdb.SearchResult, error) { return c.Client.RelatedLists(ctx, movieID, page, limit) })
}

// WatchedMovies executes the SDK operation under the CLI authentication policy.
func (c *Client) WatchedMovies(ctx context.Context) ([]map[string]any, error) {
	return call(c, ctx, true, func() ([]map[string]any, error) { return c.Client.WatchedMovies(ctx) })
}

// WantMovies executes the SDK operation under the CLI authentication policy.
func (c *Client) WantMovies(ctx context.Context) ([]map[string]any, error) {
	return call(c, ctx, true, func() ([]map[string]any, error) { return c.Client.WantMovies(ctx) })
}

// Mark executes the SDK operation under the CLI authentication policy.
func (c *Client) Mark(ctx context.Context, movieID, status string, score int, content string) (map[string]any, error) {
	return call(c, ctx, true, func() (map[string]any, error) { return c.Client.Mark(ctx, movieID, status, score, content) })
}

// Unmark executes the SDK operation under the CLI authentication policy.
func (c *Client) Unmark(ctx context.Context, movieID string) (bool, error) {
	return call(c, ctx, true, func() (bool, error) { return c.Client.Unmark(ctx, movieID) })
}

// Collected executes the SDK operation under the CLI authentication policy.
func (c *Client) Collected(ctx context.Context, kind string) ([]map[string]any, error) {
	return call(c, ctx, true, func() ([]map[string]any, error) { return c.Client.Collected(ctx, kind) })
}

// RecentViewed executes the SDK operation under the CLI authentication policy.
func (c *Client) RecentViewed(ctx context.Context) ([]map[string]any, error) {
	return call(c, ctx, true, func() ([]map[string]any, error) { return c.Client.RecentViewed(ctx) })
}

// RefreshTagTaxonomy executes the SDK operation under the CLI authentication policy.
func (c *Client) RefreshTagTaxonomy(ctx context.Context, zone string) (*tags.Doc, error) {
	return call(c, ctx, false, func() (*tags.Doc, error) { return c.Client.RefreshTagTaxonomy(ctx, zone) })
}

// LoadOrRefreshTaxonomy executes the SDK operation under the CLI authentication policy.
func (c *Client) LoadOrRefreshTaxonomy(ctx context.Context, zone string, force bool) (*tags.Doc, string, error) {
	var doc *tags.Doc
	var path string
	err := c.execute(ctx, func() error {
		var err error
		doc, path, err = c.Client.LoadOrRefreshTaxonomy(ctx, zone, force)
		return err
	})
	return doc, path, err
}

// ResolveTags executes the SDK operation under the CLI authentication policy.
func (c *Client) ResolveTags(ctx context.Context, refs []string, zone string) ([]string, error) {
	return call(c, ctx, false, func() ([]string, error) { return c.Client.ResolveTags(ctx, refs, zone) })
}

// Browse executes the SDK operation under the CLI authentication policy.
func (c *Client) Browse(ctx context.Context, opt javdb.BrowseOptions) (javdb.SearchResult, error) {
	return call(c, ctx, false, func() (javdb.SearchResult, error) { return c.Client.Browse(ctx, opt) })
}

// ResolveMovieIDExact executes the SDK operation under the CLI authentication policy.
func (c *Client) ResolveMovieIDExact(ctx context.Context, number string) (string, error) {
	return call(c, ctx, false, func() (string, error) { return c.Client.ResolveMovieIDExact(ctx, number) })
}

// RankingsMovies executes the SDK operation under the CLI authentication policy.
func (c *Client) RankingsMovies(ctx context.Context, type_, period string) (javdb.SearchResult, error) {
	return call(c, ctx, false, func() (javdb.SearchResult, error) { return c.Client.RankingsMovies(ctx, type_, period) })
}

// RankingsActors executes the SDK operation under the CLI authentication policy.
func (c *Client) RankingsActors(ctx context.Context, period string) (javdb.SearchResult, error) {
	return call(c, ctx, false, func() (javdb.SearchResult, error) { return c.Client.RankingsActors(ctx, period) })
}

// RankingsPlayback executes the SDK operation under the CLI authentication policy.
func (c *Client) RankingsPlayback(ctx context.Context, filterBy, period string) (javdb.SearchResult, error) {
	return call(c, ctx, false, func() (javdb.SearchResult, error) { return c.Client.RankingsPlayback(ctx, filterBy, period) })
}

// Top250 executes the SDK operation under the CLI authentication policy.
func (c *Client) Top250(ctx context.Context, zone, year string, startRank, page, limit int, ignoreWatched bool) (javdb.SearchResult, error) {
	return call(c, ctx, true, func() (javdb.SearchResult, error) {
		return c.Client.Top250(ctx, zone, year, startRank, page, limit, ignoreWatched)
	})
}

// Search executes the SDK operation under the CLI authentication policy.
func (c *Client) Search(ctx context.Context, keyword string, opt javdb.SearchOptions) (javdb.SearchResult, error) {
	return call(c, ctx, false, func() (javdb.SearchResult, error) { return c.Client.Search(ctx, keyword, opt) })
}

// EntityMovies executes the SDK operation under the CLI authentication policy.
func (c *Client) EntityMovies(ctx context.Context, kind, entityID string, opt javdb.EntityMoviesOptions) (javdb.SearchResult, error) {
	return call(c, ctx, false, func() (javdb.SearchResult, error) { return c.Client.EntityMovies(ctx, kind, entityID, opt) })
}

// ResolveEntity executes the SDK operation under the CLI authentication policy.
func (c *Client) ResolveEntity(ctx context.Context, kind, ref, zone string) (string, error) {
	return call(c, ctx, false, func() (string, error) { return c.Client.ResolveEntity(ctx, kind, ref, zone) })
}

// EntityDetail executes the SDK operation under the CLI authentication policy.
func (c *Client) EntityDetail(ctx context.Context, kind, id string) (map[string]any, error) {
	return call(c, ctx, false, func() (map[string]any, error) { return c.Client.EntityDetail(ctx, kind, id) })
}

// AllEntityMovies executes the SDK operation under the CLI authentication policy.
func (c *Client) AllEntityMovies(ctx context.Context, kind, entityID string, opt javdb.EntityMoviesOptions, maxPages int) ([]map[string]any, error) {
	return call(c, ctx, false, func() ([]map[string]any, error) { return c.Client.AllEntityMovies(ctx, kind, entityID, opt, maxPages) })
}

// MovieDetail executes the SDK operation under the CLI authentication policy.
func (c *Client) MovieDetail(ctx context.Context, movieID string) (map[string]any, error) {
	return call(c, ctx, false, func() (map[string]any, error) { return c.Client.MovieDetail(ctx, movieID) })
}

// MovieMagnets executes the SDK operation under the CLI authentication policy.
func (c *Client) MovieMagnets(ctx context.Context, movieID string) ([]map[string]any, error) {
	return call(c, ctx, false, func() ([]map[string]any, error) { return c.Client.MovieMagnets(ctx, movieID) })
}

// MovieComments executes the SDK operation under the CLI authentication policy.
func (c *Client) MovieComments(ctx context.Context, movieID string, page, limit int) ([]map[string]any, error) {
	return call(c, ctx, false, func() ([]map[string]any, error) { return c.Client.MovieComments(ctx, movieID, page, limit) })
}

// ResolveMovieID executes the SDK operation under the CLI authentication policy.
func (c *Client) ResolveMovieID(ctx context.Context, number string) (string, error) {
	return call(c, ctx, false, func() (string, error) { return c.Client.ResolveMovieID(ctx, number) })
}
