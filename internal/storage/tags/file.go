package tags

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
)

// Path 返回 ~/.javdb-cli/tags-{zone}.json。
func Path(zone string) (string, error) {
	return paths.TagTaxonomyPath(zone)
}

// Load 读取 taxonomy；文件不存在或文档没有 categories 时保持原有 nil 语义。
func Load(path string) (*Doc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var doc Doc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Categories == nil {
		return nil, nil
	}
	return &doc, nil
}

// Save 以公开 catalog 的 0644 权限写入 pretty JSON。
func Save(path string, doc *Doc) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
