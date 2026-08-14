package javdb

import (
	"sort"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi"
)

// Re-export magnet helpers for SDK callers.
func FilterMagnets(magnets []map[string]any, cnsub, hd bool, minSize int) []map[string]any {
	return appapi.FilterMagnets(magnets, cnsub, hd, minSize)
}

func PickBestMagnet(magnets []map[string]any) map[string]any {
	return appapi.PickBestMagnet(magnets)
}

func MagnetURI(m map[string]any) string {
	return appapi.MagnetURI(m)
}

// RankMagnets 按优先规则（cnsub > hd > size > files_count）稳定排序磁力列表
// 并按 count 截取。count <= 0 返回全部排序结果；count > 0 返回前 count 项。
// 不修改输入切片。
func RankMagnets(magnets []map[string]any, count int) []map[string]any {
	if len(magnets) == 0 {
		return magnets
	}
	ranked := make([]map[string]any, len(magnets))
	copy(ranked, magnets)
	sort.SliceStable(ranked, func(i, j int) bool {
		return appapi.MagnetBetter(ranked[i], ranked[j])
	})
	if count > 0 && count < len(ranked) {
		ranked = ranked[:count]
	}
	return ranked
}
