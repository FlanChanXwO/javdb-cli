package assets

import (
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
		got, err := parseAssetSelector(tc.input)
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
