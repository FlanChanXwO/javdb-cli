// Package entity 提供实体用例与命名实体投影。
//
// Execute 统一实体解析、tag 解析、单页/全部页查询、影片过滤与实体 metadata 行为；
// 不创建 Cobra command、不写输出、不编码 JSON。
package entity

import (
	"context"
	"fmt"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/movie"
	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// Options 是实体查询的 flag 投影。
type Options struct {
	Zone       string
	Sort       string
	Order      string
	Page       int
	Limit      int
	TagRefs    []string
	Main       []string
	AllPages   bool
	HasMagnets bool
}

// Result 是实体查询结果。
type Result struct {
	Entity   map[string]any
	EntityID string
	Movies   []map[string]any
}

// NamedRow 是命名实体列表单行的结构化投影。
type NamedRow struct {
	ID       string
	Name     string
	Count    any
	HasCount bool
}

// Line 返回命名实体行文本 `id\tname[\tcount]`（不含尾随换行）。
// 纯字符串投影，不执行 IO；空列表文案由命令负责。
func (r NamedRow) Line() string {
	if r.HasCount {
		return r.ID + "\t" + r.Name + "\t" + fmt.Sprint(r.Count)
	}
	return r.ID + "\t" + r.Name
}

// Execute 执行一个实体命令的查询用例。kind 是 actor/series/maker/director/code/list。
func Execute(ctx context.Context, client *javdb.Client, kind, ref string, options Options) (Result, error) {
	eid, err := client.ResolveEntity(ctx, kind, ref, options.Zone)
	if err != nil {
		return Result{}, fmt.Errorf("%s failed: %w", kind, err)
	}
	var tagIDs []string
	if len(options.TagRefs) > 0 {
		tagIDs, err = client.ResolveTags(ctx, options.TagRefs, options.Zone)
		if err != nil {
			return Result{}, fmt.Errorf("%s failed: %w", kind, err)
		}
	}
	opt := javdb.EntityMoviesOptions{
		Zone: options.Zone, Page: options.Page, Limit: options.Limit,
		Sort: options.Sort, Order: options.Order, Main: options.Main, Tags: tagIDs,
	}
	var movies []map[string]any
	if options.AllPages {
		movies, err = client.AllEntityMovies(ctx, kind, eid, opt, 50)
		if err != nil {
			return Result{}, fmt.Errorf("%s failed: %w", kind, err)
		}
	} else {
		res, err := client.EntityMovies(ctx, kind, eid, opt)
		if err != nil {
			return Result{}, fmt.Errorf("%s failed: %w", kind, err)
		}
		movies = res.Movies()
	}
	if options.HasMagnets {
		movies = movie.FilterHasMagnets(movies)
	}
	var meta map[string]any
	if m, err := client.EntityDetail(ctx, kind, eid); err == nil {
		meta = m
	} else {
		meta = map[string]any{"id": eid}
	}
	return Result{Entity: meta, EntityID: eid, Movies: movies}, nil
}

// ProjectNamed 将命名实体记录投影为 NamedRow；name_zht 优先，其次 name。
func ProjectNamed(item map[string]any) NamedRow {
	name := scalar.String(item["name_zht"])
	if name == "" {
		name = scalar.String(item["name"])
	}
	count := item["videos_count"]
	if count == nil {
		count = item["movies_count"]
	}
	return NamedRow{
		ID:       scalar.String(item["id"]),
		Name:     name,
		Count:    count,
		HasCount: count != nil,
	}
}

// ProjectNamedAll 将命名实体记录列表投影为 NamedRow 列表。
func ProjectNamedAll(items []map[string]any) []NamedRow {
	out := make([]NamedRow, 0, len(items))
	for _, item := range items {
		out = append(out, ProjectNamed(item))
	}
	return out
}
