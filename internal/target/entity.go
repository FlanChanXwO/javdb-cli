// Package target 只保留六类实体命令共享的查询用例。
//
// Execute 统一实体解析、tag 解析、单页/全部页查询、影片过滤与实体 metadata 行为；
// 不创建 Cobra command、不写输出、不编码 JSON。命名实体投影位于 internal/display。
package target

import (
	"context"

	"github.com/FlanChanXwO/javdb-cli/internal/display"
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

// Client is the public SDK capability used by entity queries.
type Client interface {
	ResolveEntity(context.Context, string, string, string) (string, error)
	ResolveTags(context.Context, []string, string) ([]string, error)
	AllEntityMovies(context.Context, string, string, javdb.EntityMoviesOptions, int) ([]map[string]any, error)
	EntityMovies(context.Context, string, string, javdb.EntityMoviesOptions) (javdb.SearchResult, error)
	EntityDetail(context.Context, string, string) (map[string]any, error)
}

// Execute 执行一个实体命令的查询用例。kind 是 actor/series/maker/director/code/list。
func Execute(ctx context.Context, client Client, kind, ref string, options Options) (Result, error) {
	eid, err := client.ResolveEntity(ctx, kind, ref, options.Zone)
	if err != nil {
		return Result{}, err
	}
	var tagIDs []string
	if len(options.TagRefs) > 0 {
		tagIDs, err = client.ResolveTags(ctx, options.TagRefs, options.Zone)
		if err != nil {
			return Result{}, err
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
			return Result{}, err
		}
	} else {
		res, err := client.EntityMovies(ctx, kind, eid, opt)
		if err != nil {
			return Result{}, err
		}
		movies = res.Movies()
	}
	if options.HasMagnets {
		movies = display.FilterMoviesWithMagnets(movies)
	}
	var meta map[string]any
	if m, err := client.EntityDetail(ctx, kind, eid); err == nil {
		meta = m
	} else {
		meta = map[string]any{"id": eid}
	}
	return Result{Entity: meta, EntityID: eid, Movies: movies}, nil
}
