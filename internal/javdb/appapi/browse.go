package appapi

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/storage/tags"
)

// TagsRaw fetches GET /api/v2/tags?type={zone digit} categories list under "tags".
func (c *Client) TagsRaw(zone, lang string) ([]map[string]any, error) {
	z, ok := Zones[zone]
	if !ok {
		return nil, fmt.Errorf("invalid zone %s", zone)
	}
	prev := c.lang
	if lang != "" {
		c.lang = lang
	}
	defer func() { c.lang = prev }()
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v2/tags", map[string]string{"type": strconv.Itoa(z)}, &data); err != nil {
		return nil, err
	}
	return decodeObjectArray(data["tags"]), nil
}

// RefreshTagTaxonomy fetches EN+ZH and writes tags-{zone}.json.
func (c *Client) RefreshTagTaxonomy(zone string) (*tags.Doc, error) {
	z, ok := Zones[zone]
	if !ok {
		return nil, fmt.Errorf("invalid zone %s", zone)
	}
	enCats, err := c.TagsRaw(zone, "en")
	if err != nil {
		return nil, err
	}
	zhCats, err := c.TagsRaw(zone, "zh-TW")
	if err != nil {
		return nil, err
	}
	zhByID := map[string]string{}
	zhCatNames := map[string]string{}
	for _, cat := range zhCats {
		cid := anyStr(cat["category_id"])
		zhCatNames[cid] = anyStr(cat["category"])
		for _, t := range asMapSlice(cat["tags"]) {
			tid := anyStr(t["id"])
			if tid != "" {
				zhByID[tid] = anyStr(t["name"])
			}
		}
	}
	doc := &tags.Doc{Zone: zone, Type: z}
	for _, cat := range enCats {
		cid := anyStr(cat["category_id"])
		var tagsOut []tags.Tag
		for _, t := range asMapSlice(cat["tags"]) {
			tid := anyStr(t["id"])
			if tid == "" {
				continue
			}
			tagsOut = append(tagsOut, tags.Tag{
				ID: tid, NameEN: anyStr(t["name"]), NameZH: zhByID[tid],
			})
		}
		doc.Categories = append(doc.Categories, tags.Category{
			ID: cid, NameEN: anyStr(cat["category"]), NameZH: zhCatNames[cid], Tags: tagsOut,
		})
	}
	path, err := tags.Path(zone)
	if err != nil {
		return nil, err
	}
	if err := tags.Save(path, doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// LoadOrRefreshTaxonomy returns disk doc or refreshes.
func (c *Client) LoadOrRefreshTaxonomy(zone string, force bool) (*tags.Doc, string, error) {
	path, err := tags.Path(zone)
	if err != nil {
		return nil, "", err
	}
	if !force {
		doc, err := tags.Load(path)
		if err != nil {
			return nil, path, err
		}
		if doc != nil {
			return doc, path, nil
		}
	}
	doc, err := c.RefreshTagTaxonomy(zone)
	return doc, path, err
}

// ResolveTags maps free-form refs to ids for a zone.
func (c *Client) ResolveTags(refs []string, zone string) ([]string, error) {
	doc, _, err := c.LoadOrRefreshTaxonomy(zone, false)
	if err != nil {
		return nil, err
	}
	return tags.ResolveRefs(refs, tags.AliasMap(doc))
}

// BrowseOptions for GET /api/v1/movies/tags category page.
type BrowseOptions struct {
	Zone   string
	Main   []string
	TagIDs []string // already resolved ids
	Year   string
	Month  string
	Sort   string
	Order  string
	Page   int
	Limit  int
}

// Browse lists movies by tag filter mask.
func (c *Client) Browse(opt BrowseOptions) (SearchResult, error) {
	if opt.Zone == "" {
		opt.Zone = "censored"
	}
	if opt.Page <= 0 {
		opt.Page = 1
	}
	if opt.Limit <= 0 {
		opt.Limit = 20
	}
	if opt.Sort == "" {
		opt.Sort = "hit"
	}
	if opt.Order == "" {
		opt.Order = "desc"
	}
	mask, err := BuildTagFilter(opt.Zone, opt.Main, opt.TagIDs, opt.Year, opt.Month)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"filter_by": mask,
		"sort_by":   opt.Sort,
		"order_by":  opt.Order,
		"page":      strconv.Itoa(opt.Page),
		"limit":     strconv.Itoa(opt.Limit),
	}
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/movies/tags", params, &data); err != nil {
		return nil, err
	}
	return SearchResult(data), nil
}

func asMapSlice(v any) []map[string]any {
	switch t := v.(type) {
	case []map[string]any:
		return t
	case []any:
		out := make([]map[string]any, 0, len(t))
		for _, x := range t {
			if m, ok := x.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		// try re-marshal
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		var out []map[string]any
		_ = json.Unmarshal(b, &out)
		return out
	}
}
