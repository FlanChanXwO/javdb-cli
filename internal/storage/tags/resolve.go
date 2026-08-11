package tags

import (
	"fmt"
	"strings"
)

// AliasMap 将大小写折叠后的 id、英文名、中文名映射到 canonical id。
func AliasMap(doc *Doc) map[string]string {
	aliases := make(map[string]string)
	if doc == nil {
		return aliases
	}
	for _, category := range doc.Categories {
		for _, tag := range category.Tags {
			if tag.ID == "" {
				continue
			}
			aliases[strings.ToLower(tag.ID)] = tag.ID
			if english := strings.ToLower(strings.TrimSpace(tag.NameEN)); english != "" {
				aliases[english] = tag.ID
			}
			if chinese := strings.ToLower(strings.TrimSpace(tag.NameZH)); chinese != "" {
				aliases[chinese] = tag.ID
			}
		}
	}
	return aliases
}

// ResolveRefs 将自由输入的 tag 引用解析为 id。
func ResolveRefs(refs []string, aliases map[string]string) ([]string, error) {
	resolved := make([]string, 0, len(refs))
	for _, ref := range refs {
		key := strings.ToLower(strings.TrimSpace(ref))
		if key == "" {
			continue
		}
		id, ok := aliases[key]
		if !ok {
			return nil, fmt.Errorf("unknown tag %q (run: javdb tags --refresh)", ref)
		}
		resolved = append(resolved, id)
	}
	return resolved, nil
}
