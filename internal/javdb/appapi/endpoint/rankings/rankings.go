package rankings

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/client"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
)

// RankingsEndpoint 提供排行与 TOP250 capability。
type RankingsEndpoint struct {
	c *client.Client
}

// NewRankings 用共享 transport 构造 rankings capability。
func NewRankings(c *client.Client) *RankingsEndpoint {
	return &RankingsEndpoint{c: c}
}

// RankingPeriod maps CLI day/week/month → API daily/weekly/monthly for rankings.
func RankingPeriod(period string) string {
	switch period {
	case "day":
		return "daily"
	case "week":
		return "weekly"
	case "month":
		return "monthly"
	default:
		return period
	}
}

// ActorPeriod maps CLI day/week/month → API daily/weekly/monthly for actor rankings.
// Deprecated: Use RankingPeriod instead.
func ActorPeriod(period string) string {
	return RankingPeriod(period)
}

func normalizeZone(zone string) string {
	if z, ok := model.Zones[zone]; ok {
		return strconv.Itoa(z)
	}
	return zone
}

// BuildTop250Params builds query for GET /api/v1/movies/top.
// year wins over zone when both set. Empty both → type=all.
func BuildTop250Params(zone, year string, startRank, page, limit int, ignoreWatched bool) (map[string]string, error) {
	if startRank <= 0 {
		startRank = 1
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	var t, tv string
	switch {
	case year != "":
		t, tv = "year", year
	case zone != "":
		z, ok := model.Zones[zone]
		if !ok {
			return nil, fmt.Errorf("zone must be one of censored|uncensored|western|fc2")
		}
		t, tv = "video_type", strconv.Itoa(z)
	default:
		t, tv = "all", "all"
	}
	iw := "false"
	if ignoreWatched {
		iw = "true"
	}
	return map[string]string{
		"type":           t,
		"type_value":     tv,
		"start_rank":     strconv.Itoa(startRank),
		"page":           strconv.Itoa(page),
		"limit":          strconv.Itoa(limit),
		"ignore_watched": iw,
	}, nil
}

// RankingsMovies GET /api/v1/rankings
func (e *RankingsEndpoint) RankingsMovies(type_, period string) (model.SearchResult, error) {
	var data map[string]json.RawMessage
	if err := e.c.GetJSON("/api/v1/rankings", map[string]string{
		"type": normalizeZone(type_), "period": RankingPeriod(period),
	}, &data); err != nil {
		return nil, err
	}
	return model.SearchResult(data), nil
}

// RankingsActors GET /api/v1/rankings/actors — period accepts day|week|month or daily|weekly|monthly.
func (e *RankingsEndpoint) RankingsActors(period string) (model.SearchResult, error) {
	var data map[string]json.RawMessage
	if err := e.c.GetJSON("/api/v1/rankings/actors", map[string]string{
		"type": RankingPeriod(period),
	}, &data); err != nil {
		return nil, err
	}
	return model.SearchResult(data), nil
}

// RankingsPlayback GET /api/v1/rankings/playback
func (e *RankingsEndpoint) RankingsPlayback(filterBy, period string) (model.SearchResult, error) {
	var data map[string]json.RawMessage
	if err := e.c.GetJSON("/api/v1/rankings/playback", map[string]string{
		"filter_by": normalizeZone(filterBy), "period": RankingPeriod(period),
	}, &data); err != nil {
		return nil, err
	}
	return model.SearchResult(data), nil
}

// Top250 GET /api/v1/movies/top (auth required).
func (e *RankingsEndpoint) Top250(zone, year string, startRank, page, limit int, ignoreWatched bool) (model.SearchResult, error) {
	params, err := BuildTop250Params(zone, year, startRank, page, limit, ignoreWatched)
	if err != nil {
		return nil, err
	}
	var data map[string]json.RawMessage
	if err := e.c.GetJSON("/api/v1/movies/top", params, &data); err != nil {
		return nil, err
	}
	return model.SearchResult(data), nil
}
