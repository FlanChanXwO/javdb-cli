package main

import "strings"

// documentationOnlyPolicy 是 CI 路径分类的唯一白名单。它只涵盖不会影响运行产物的
// 文档、Agent 指令和协作元数据；新增路径时必须先排除构建、发布或执行行为影响。
var documentationOnlyPolicy = documentationPathPolicy{
	rootMarkdownPrefixes: []string{"README.", "CONTRIBUTING.", "CHANGELOG."},
	exactPaths: map[string]struct{}{
		"README.md":                        {},
		"AGENTS.md":                        {},
		"CLAUDE.md":                        {},
		"CONTRIBUTING.md":                  {},
		"CHANGELOG.md":                     {},
		".gitignore":                       {},
		".pre-commit-config.yaml":          {},
		".github/CODEOWNERS":               {},
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
