package assets

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// parseAssetSelector 解析 1-based 资产选择器。
// 支持单个编号("1")、空格/逗号分隔的多编号("1 3 5"、"1,3,5")与闭区间("1-4"),
// 可混用("1,3-5"、"1 3-5")。空输入返回 nil 表示不选择(即全部)。
// 重叠编号去重,结果按编号升序,即当前列表顺序。
// 编号不是长期资产 ID,只是过滤后列表的当前位置。
// assetCount 是过滤后资产总数:range 在展开前先校验上界(计划 #9),
// 防止 "1-1000000000" 在知道资产数量前分配巨大 slice。
func parseAssetSelector(input string, assetCount int) ([]int, error) {
	tokens := strings.FieldsFunc(input, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' })
	if len(tokens) == 0 {
		return nil, nil
	}
	seen := make(map[int]bool)
	for _, token := range tokens {
		nums, err := parseSelectorToken(token, assetCount)
		if err != nil {
			return nil, err
		}
		for _, n := range nums {
			seen[n] = true
		}
	}
	selected := make([]int, 0, len(seen))
	for n := range seen {
		selected = append(selected, n)
	}
	sort.Ints(selected)
	return selected, nil
}

// parseSelectorToken 解析单个编号或闭区间 token。
// 解析 start/end 后先结合 assetCount 校验,再展开(计划 #9)。
func parseSelectorToken(token string, assetCount int) ([]int, error) {
	invalid := fmt.Errorf("invalid selector %q", token)
	if before, after, found := strings.Cut(token, "-"); found {
		start, errStart := strconv.Atoi(before)
		end, errEnd := strconv.Atoi(after)
		if errStart != nil || errEnd != nil || start < 1 || end < start {
			return nil, invalid
		}
		// 结合资产数校验后再展开(计划 #9):
		// 禁止在不知道资产数量时展开巨大 range。
		if start > assetCount {
			return nil, fmt.Errorf("asset number %d out of range (1-%d)", start, assetCount)
		}
		if end > assetCount {
			return nil, fmt.Errorf("asset number %d out of range (1-%d)", end, assetCount)
		}
		nums := make([]int, 0, end-start+1)
		for n := start; n <= end; n++ {
			nums = append(nums, n)
		}
		return nums, nil
	}
	n, err := strconv.Atoi(token)
	if err != nil || n < 1 {
		return nil, invalid
	}
	return []int{n}, nil
}
