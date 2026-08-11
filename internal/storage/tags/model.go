// Package tags 持久化和解析 JavDB tag taxonomy cache。
package tags

// Doc 是磁盘 taxonomy 文档的结构。
type Doc struct {
	Zone       string     `json:"zone"`
	Type       int        `json:"type"`
	Categories []Category `json:"categories"`
}

// Category 是 taxonomy 中的一组 tag。
type Category struct {
	ID     string `json:"id"`
	NameEN string `json:"name_en"`
	NameZH string `json:"name_zh"`
	Tags   []Tag  `json:"tags"`
}

// Tag 是 taxonomy 中的单个 tag。
type Tag struct {
	ID     string `json:"id"`
	NameEN string `json:"name_en"`
	NameZH string `json:"name_zh"`
}
