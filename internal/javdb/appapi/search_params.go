package appapi

import "strconv"

// Zone digit for movie_type / filter_by masks.
var Zones = map[string]int{
	"censored":   0,
	"uncensored": 1,
	"western":    2,
	"fc2":        3,
}

// SearchOptions configures GET /api/v2/search.
//
// Zone: "censored", "uncensored", "western", "fc2", or "all"/"" to omit movie_type.
type SearchOptions struct {
	Page     int
	Limit    int // 0 = omit
	Zone     string
	Sort     string // movie_sort_by
	FilterBy string // movie_filter_by
	Type     string // result dimension
}

// BuildSearchParams returns query params for /api/v2/search (public params merged by Client).
func BuildSearchParams(keyword string, opt SearchOptions) map[string]string {
	if opt.Page <= 0 {
		opt.Page = 1
	}
	p := map[string]string{
		"q":    keyword,
		"page": strconv.Itoa(opt.Page),
	}
	if opt.Zone != "" && opt.Zone != "all" {
		if z, ok := Zones[opt.Zone]; ok {
			p["movie_type"] = strconv.Itoa(z)
		}
	}
	if opt.Sort != "" {
		p["movie_sort_by"] = opt.Sort
	}
	if opt.FilterBy != "" {
		p["movie_filter_by"] = opt.FilterBy
	}
	if opt.Type != "" {
		p["type"] = opt.Type
	}
	if opt.Limit > 0 {
		p["limit"] = strconv.Itoa(opt.Limit)
	}
	return p
}
