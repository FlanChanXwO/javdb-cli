package lists

import (
	"fmt"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
)

// listEnvelope 将 API 返回的合集原始项投影为统一的列表信封。
func listEnvelope(item map[string]any) (pipeline.Envelope, error) {
	id := display(item["id"])
	if id == "" {
		return pipeline.Envelope{}, fmt.Errorf("list has no id")
	}
	ref := display(item["name"])
	if ref == "" {
		ref = id
	}
	return pipeline.New(pipeline.KindList, ref, id).WithData(map[string]any{"list": item}), nil
}
