package javdb

import (
	"context"

	"github.com/FlanChanXwO/javdb-cli/internal/appapi"
)

// ActorPeriod re-export.
func ActorPeriod(period string) string { return appapi.ActorPeriod(period) }

// RankingsMovies fetches movie rankings.
func (c *Client) RankingsMovies(ctx context.Context, type_, period string) (SearchResult, error) {
	_ = ctx
	return c.api.RankingsMovies(type_, period)
}

// RankingsActors fetches actor rankings (period already daily/weekly/monthly or use ActorPeriod).
func (c *Client) RankingsActors(ctx context.Context, period string) (SearchResult, error) {
	_ = ctx
	return c.api.RankingsActors(period)
}

// RankingsPlayback fetches playback rankings.
func (c *Client) RankingsPlayback(ctx context.Context, filterBy, period string) (SearchResult, error) {
	_ = ctx
	return c.api.RankingsPlayback(filterBy, period)
}

// Top250 fetches TOP250 (auth).
func (c *Client) Top250(ctx context.Context, zone, year string, startRank, page, limit int, ignoreWatched bool) (SearchResult, error) {
	_ = ctx
	return c.api.Top250(zone, year, startRank, page, limit, ignoreWatched)
}
