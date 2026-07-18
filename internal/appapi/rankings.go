package appapi

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// ActorPeriod maps CLI day/week/month → API daily/weekly/monthly for actor rankings.
func ActorPeriod(period string) string {
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
		z, ok := Zones[zone]
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
		"type":            t,
		"type_value":      tv,
		"start_rank":      strconv.Itoa(startRank),
		"page":            strconv.Itoa(page),
		"limit":           strconv.Itoa(limit),
		"ignore_watched":  iw,
	}, nil
}

// RankingsMovies GET /api/v1/rankings
func (c *Client) RankingsMovies(type_, period string) (SearchResult, error) {
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/rankings", map[string]string{
		"type": type_, "period": period,
	}, &data); err != nil {
		return nil, err
	}
	return SearchResult(data), nil
}

// RankingsActors GET /api/v1/rankings/actors — period must be daily|weekly|monthly.
func (c *Client) RankingsActors(period string) (SearchResult, error) {
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/rankings/actors", map[string]string{
		"type": period,
	}, &data); err != nil {
		return nil, err
	}
	return SearchResult(data), nil
}

// RankingsPlayback GET /api/v1/rankings/playback
func (c *Client) RankingsPlayback(filterBy, period string) (SearchResult, error) {
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/rankings/playback", map[string]string{
		"filter_by": filterBy, "period": period,
	}, &data); err != nil {
		return nil, err
	}
	return SearchResult(data), nil
}

// Top250 GET /api/v1/movies/top (auth required).
func (c *Client) Top250(zone, year string, startRank, page, limit int, ignoreWatched bool) (SearchResult, error) {
	params, err := BuildTop250Params(zone, year, startRank, page, limit, ignoreWatched)
	if err != nil {
		return nil, err
	}
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/movies/top", params, &data); err != nil {
		return nil, err
	}
	return SearchResult(data), nil
}
