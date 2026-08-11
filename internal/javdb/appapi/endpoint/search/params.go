package search

import (
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
)

// Zones 保留 endpoint 内部对共享 zone map 的便捷引用。
var Zones = model.Zones

// SearchOptions 是 model.SearchOptions 的 endpoint alias。
type SearchOptions = model.SearchOptions

// BuildSearchParams returns query params for /api/v2/search (public params merged by client).
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
