package assets

import (
	"strings"

	javdb "github.com/FlanChanXwO/javdb-cli/sdk"

	"reflect"
	"testing"
)

// selector 契约(input.md 计划 #7/#8):支持单个编号、空格/逗号分隔、闭区间,
// 可混用;编号 1-based;重叠去重并按列表顺序输出;非法输入明确报错。

func TestParseAssetSelector(t *testing.T) {
	cases := []struct {
		input   string
		want    []int
		wantErr bool
	}{
		{input: "", want: nil},
		{input: "1", want: []int{1}},
		{input: "1 3 5", want: []int{1, 3, 5}},
		{input: "1-4", want: []int{1, 2, 3, 4}},
		{input: "1,3,5", want: []int{1, 3, 5}},
		{input: "1-3,5", want: []int{1, 2, 3, 5}},
		{input: "1 3-5", want: []int{1, 3, 4, 5}},
		// 重叠区间去重,按列表顺序输出。
		{input: "1-3,2-4", want: []int{1, 2, 3, 4}},
		{input: "3,1-2", want: []int{1, 2, 3}},
		// 区间内部不允许空白:空白是编号分隔符。
		{input: "1 - 2", wantErr: true},
		{input: "0", wantErr: true},
		{input: "-1", wantErr: true},
		{input: "1-", wantErr: true},
		{input: "4-1", wantErr: true},
		{input: "foo", wantErr: true},
		{input: "1.5", wantErr: true},
	}
	for _, tc := range cases {
		got, err := parseAssetSelector(tc.input, 100)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("parseAssetSelector(%q) = %v, want error", tc.input, got)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parseAssetSelector(%q) error = %v", tc.input, err)
		}
		if tc.want == nil {
			if got != nil {
				t.Fatalf("parseAssetSelector(%q) = %v, want nil", tc.input, got)
			}
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("parseAssetSelector(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

// ---- 计划 #9:selector 防止巨型区间提前展开 ----

// 1-1000000000 会在检查实际资产数量前尝试分配巨大 slice:
// 必须先解析 start/end,结合资产数校验,再展开。
func TestParseAssetSelectorRejectsHugeRange(t *testing.T) {
	// 资产数 6:range 1-1000000000 超出范围,必须在展开前报错。
	_, err := parseAssetSelector("1-1000000000", 6)
	if err == nil {
		t.Fatal("huge range must be rejected before expansion")
	}
}

// 结合资产数校验:selectAssets 必须先校验 range 上界。
func TestSelectAssetsRejectsRangeBeyondAssets(t *testing.T) {
	assets := []javdb.MovieAsset{
		{Type: "image", URL: "a"},
		{Type: "image", URL: "b"},
	}
	descs := []string{"a", "b"}
	_, _, err := selectAssets(assets, descs, "1-1000000000")
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("error = %v, want out of range before expansion", err)
	}
}

// ---- 计划 #10:movieString 只接受 string ----
