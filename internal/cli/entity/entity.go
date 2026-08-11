// Package entity 只保留六类实体命令共享的查询用例。
//
// Execute 统一实体解析、tag 解析、单页/全部页查询、影片过滤与实体 metadata 行为；
// 不创建 Cobra command、不写输出、不编码 JSON。命名实体投影位于 internal/cli/result。
package entity

import (
	"context"
	"fmt"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
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
		movies = result.FilterMoviesWithMagnets(movies)
	}
	var meta map[string]any
	if m, err := client.EntityDetail(ctx, kind, eid); err == nil {
		meta = m
	} else {
		meta = map[string]any{"id": eid}
	}
	return Result{Entity: meta, EntityID: eid, Movies: movies}, nil
}
