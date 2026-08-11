package result

import (
	"fmt"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
)

// NamedRow 是命名实体列表单行的结构化投影。
type NamedRow struct {
	ID       string
	Name     string
	Count    any
	HasCount bool
}

// Line 返回命名实体行文本 `id\tname[\tcount]`（不含尾随换行）。
func (r NamedRow) Line() string {
	if r.HasCount {
		return r.ID + "\t" + r.Name + "\t" + fmt.Sprint(r.Count)
	}
	return r.ID + "\t" + r.Name
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
