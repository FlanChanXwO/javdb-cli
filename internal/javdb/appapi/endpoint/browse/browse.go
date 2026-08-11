package browse

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/client"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/codec"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/tags"
)

// BrowseEndpoint 提供 tag 分类浏览、taxonomy 与 tag 解析 capability。
type BrowseEndpoint struct {
	c *client.Client
}

// NewBrowse 用共享 transport 构造 browse capability。
func NewBrowse(c *client.Client) *BrowseEndpoint {
	return &BrowseEndpoint{c: c}
}

// TagsRaw fetches GET /api/v2/tags?type={zone digit} categories list under "tags".
func (e *BrowseEndpoint) TagsRaw(zone, lang string) ([]map[string]any, error) {
	z, ok := model.Zones[zone]
	if !ok {
		return nil, fmt.Errorf("invalid zone %s", zone)
	}
	prev := e.c.Language()
	if lang != "" {
		e.c.SetLanguage(lang)
	}
	defer func() { e.c.SetLanguage(prev) }()
	var data map[string]json.RawMessage
	if err := e.c.GetJSON("/api/v2/tags", map[string]string{"type": strconv.Itoa(z)}, &data); err != nil {
		return nil, err
	}
	return codec.ObjectArray(data["tags"]), nil
}

// RefreshTagTaxonomy fetches EN+ZH and writes tags-{zone}.json.
func (e *BrowseEndpoint) RefreshTagTaxonomy(zone string) (*tags.Doc, error) {
	z, ok := model.Zones[zone]
	if !ok {
		return nil, fmt.Errorf("invalid zone %s", zone)
	}
	enCats, err := e.TagsRaw(zone, "en")
	if err != nil {
		return nil, err
	}
	zhCats, err := e.TagsRaw(zone, "zh-TW")
	if err != nil {
		return nil, err
	}
	zhByID := map[string]string{}
	zhCatNames := map[string]string{}
	for _, cat := range zhCats {
		cid := scalar.String(cat["category_id"])
		zhCatNames[cid] = scalar.String(cat["category"])
		for _, t := range codec.ObjectSlice(cat["tags"]) {
			tid := scalar.String(t["id"])
			if tid != "" {
				zhByID[tid] = scalar.String(t["name"])
			}
		}
	}
	doc := &tags.Doc{Zone: zone, Type: z}
	for _, cat := range enCats {
		cid := scalar.String(cat["category_id"])
		var tagsOut []tags.Tag
		for _, t := range codec.ObjectSlice(cat["tags"]) {
			tid := scalar.String(t["id"])
			if tid == "" {
				continue
			}
			tagsOut = append(tagsOut, tags.Tag{
				ID: tid, NameEN: scalar.String(t["name"]), NameZH: zhByID[tid],
			})
		}
		doc.Categories = append(doc.Categories, tags.Category{
			ID: cid, NameEN: scalar.String(cat["category"]), NameZH: zhCatNames[cid], Tags: tagsOut,
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
func (e *BrowseEndpoint) LoadOrRefreshTaxonomy(zone string, force bool) (*tags.Doc, string, error) {
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
	doc, err := e.RefreshTagTaxonomy(zone)
	return doc, path, err
}

// ResolveTags maps free-form refs to ids for a zone.
func (e *BrowseEndpoint) ResolveTags(refs []string, zone string) ([]string, error) {
	doc, _, err := e.LoadOrRefreshTaxonomy(zone, false)
	if err != nil {
		return nil, err
	}
	return tags.ResolveRefs(refs, tags.AliasMap(doc))
}

// BrowseOptions 是 model.BrowseOptions 的 endpoint alias。
type BrowseOptions = model.BrowseOptions

// Browse lists movies by tag filter mask.
func (e *BrowseEndpoint) Browse(opt BrowseOptions) (model.SearchResult, error) {
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
	if err := e.c.GetJSON("/api/v1/movies/tags", params, &data); err != nil {
		return nil, err
	}
	return model.SearchResult(data), nil
}
