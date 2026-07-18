package tags

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/config"
)

// Doc is the on-disk taxonomy shape.
type Doc struct {
	Zone       string     `json:"zone"`
	Type       int        `json:"type"`
	Categories []Category `json:"categories"`
}

type Category struct {
	ID     string `json:"id"`
	NameEN string `json:"name_en"`
	NameZH string `json:"name_zh"`
	Tags   []Tag  `json:"tags"`
}

type Tag struct {
	ID     string `json:"id"`
	NameEN string `json:"name_en"`
	NameZH string `json:"name_zh"`
}

// Path returns ~/.javdb-cli/tags-{zone}.json
func Path(zone string) (string, error) {
	return config.TagTaxonomyPath(zone)
}

// Load reads taxonomy; nil if missing/invalid.
func Load(path string) (*Doc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var d Doc
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	if d.Categories == nil {
		return nil, nil
	}
	return &d, nil
}

// Save writes pretty JSON (public catalog; not 0600).
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

// AliasMap maps casefolded id/EN/ZH → canonical id.
func AliasMap(doc *Doc) map[string]string {
	m := make(map[string]string)
	if doc == nil {
		return m
	}
	for _, cat := range doc.Categories {
		for _, t := range cat.Tags {
			if t.ID == "" {
				continue
			}
			m[strings.ToLower(t.ID)] = t.ID
			if en := strings.ToLower(strings.TrimSpace(t.NameEN)); en != "" {
				m[en] = t.ID
			}
			if zh := strings.ToLower(strings.TrimSpace(t.NameZH)); zh != "" {
				m[zh] = t.ID
			}
		}
	}
	return m
}

// ResolveRefs maps free-form refs to ids using alias map.
func ResolveRefs(refs []string, aliases map[string]string) ([]string, error) {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		key := strings.ToLower(strings.TrimSpace(r))
		if key == "" {
			continue
		}
		id, ok := aliases[key]
		if !ok {
			return nil, fmt.Errorf("unknown tag %q (run: javdb tags --refresh)", r)
		}
		out = append(out, id)
	}
	return out, nil
}
