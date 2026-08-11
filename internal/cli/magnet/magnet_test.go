package magnet

import (
	"reflect"
	"testing"
)

func TestProjectUsesNameOrTitle(t *testing.T) {
	row := Project(map[string]any{
		"name": "N", "size": float64(2048), "cnsub": true, "hd": true,
		"created_at": "2026-01-02T03:04:05Z", "hash": "ABC",
	})
	if row.Name != "N" || row.Size != "2.0GB" || row.CreatedAt != "2026-01-02" || row.Hash != "ABC" {
		t.Fatalf("row = %+v", row)
	}
	if !reflect.DeepEqual(row.Flags, []string{"cnsub", "hd"}) {
		t.Fatalf("flags = %v", row.Flags)
	}
	// 无 name 时降级到 title。
	row = Project(map[string]any{"title": "T", "size": float64(64), "created_at": "2026-01-02", "hash": "DEF"})
	if row.Name != "T" || row.Size != "64MB" {
		t.Fatalf("row = %+v", row)
	}
	if len(row.Flags) != 0 {
		t.Fatalf("flags = %v", row.Flags)
	}
}

func TestProjectDateTruncatesBeyondTenChars(t *testing.T) {
	row := Project(map[string]any{"name": "N", "created_at": "2026-01-02T03:04:05.000Z", "hash": "H"})
	if row.CreatedAt != "2026-01-02" {
		t.Fatalf("created_at = %q", row.CreatedAt)
	}
}

func TestProjectAll(t *testing.T) {
	rows := ProjectAll([]map[string]any{
		{"name": "N1", "size": float64(1024), "hash": "A"},
		{"title": "N2", "size": float64(1), "hash": "B"},
	})
	if len(rows) != 2 || rows[0].Size != "1.0GB" || rows[1].Size != "1MB" {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{float64(2048), "2.0GB"},
		{float64(64), "64MB"},
		{float64(1024), "1.0GB"},
		{"1GB", "1GB"},    // 非数字字符串降级展示原值
		{nil, ""},         // 缺失字段 → 空
		{float64(0), "0"}, // 0 → "0"
	}
	for _, tc := range cases {
		if got := FormatSize(tc.in); got != tc.want {
			t.Fatalf("FormatSize(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFlagsTruthy(t *testing.T) {
	if got := Flags(map[string]any{"cnsub": true, "hd": false}); !reflect.DeepEqual(got, []string{"cnsub"}) {
		t.Fatalf("flags = %v", got)
	}
	if got := Flags(map[string]any{"cnsub": float64(1), "hd": "1"}); !reflect.DeepEqual(got, []string{"cnsub", "hd"}) {
		t.Fatalf("flags = %v", got)
	}
}
