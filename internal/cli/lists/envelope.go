package lists

import (
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
)

// listEnvelope 将 API 返回的合集原始项投影为统一的列表信封。
func listEnvelope(item map[string]any) (record.Envelope, error) {
	id := display(item["id"])
	if id == "" {
		return record.Envelope{}, i18n.Errorf("lists.missing_id")
	}
	ref := display(item["name"])
	if ref == "" {
		ref = id
	}
	return record.New(record.KindList, ref, id).WithData(map[string]any{"list": item}), nil
}
