package model

import (
	"encoding/json"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/codec"
)

// SearchResult 是 /api/v2/search 的宽松 data payload。
// 实际存在的 key 由 Type 决定（movies/codes/series/actors/makers/directors/lists）。
type SearchResult map[string]json.RawMessage

// Movies 解码 movies 数组；字段缺失、null 或非法数组返回 nil。
func (r SearchResult) Movies() []map[string]any {
	return codec.ObjectArray(r["movies"])
}

// Named 解码指定的命名维度数组（codes、series、actors 等）。
func (r SearchResult) Named(key string) []map[string]any {
	return codec.ObjectArray(r[key])
}
