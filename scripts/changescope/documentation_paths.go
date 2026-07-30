package main

import "strings"

// documentationOnlyPolicy 是 CI 路径分类的唯一白名单。它故意只涵盖不会影响
// 运行产物或门禁行为的面向用户文档、维护文档与 Agent 技能；新增路径时应先评估其
// 是否可能改变构建、发布或执行行为，不能仅因为文件使用 Markdown 就加入此处。
var documentationOnlyPolicy = documentationPathPolicy{
	rootMarkdownPrefixes: []string{"README.", "CONTRIBUTING.", "CHANGELOG."},
	exactPaths: map[string]struct{}{
		"README.md":                        {},
		"CONTRIBUTING.md":                  {},
		"CHANGELOG.md":                     {},
		".github/PULL_REQUEST_TEMPLATE.md": {},
	},
	directoryPrefixes: []string{
		"docs/",
		"changelog/",
		"skills/",
		".agents/skills/",
		".github/ISSUE_TEMPLATE/",
	},
}

type documentationPathPolicy struct {
	rootMarkdownPrefixes []string
	exactPaths           map[string]struct{}
	directoryPrefixes    []string
}

func isApprovedDocumentationPath(path string) bool {
	if _, ok := documentationOnlyPolicy.exactPaths[path]; ok {
		return true
	}
	for _, prefix := range documentationOnlyPolicy.rootMarkdownPrefixes {
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, ".md") {
			return true
		}
	}
	for _, prefix := range documentationOnlyPolicy.directoryPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}
