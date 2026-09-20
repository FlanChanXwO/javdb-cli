// Command changescope classifies a Git diff for GitHub Actions CI routing.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	base := flag.String("base", "", "base Git commit")
	head := flag.String("head", "", "head Git commit")
	githubOutput := flag.String("github-output", "", "GitHub Actions output file")
	flag.Parse()

	scope, reason, err := classify(*base, *head)
	if err != nil {
		fmt.Fprintf(os.Stderr, "classify change scope: %v\n", err)
		os.Exit(1)
	}
	if err := writeOutput(*githubOutput, scope); err != nil {
		fmt.Fprintf(os.Stderr, "write change scope output: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, reason)
	fmt.Printf("docs_only=%t\n", scope.DocsOnly)
	fmt.Printf("quality_required=%t\n", scope.QualityRequired)
	fmt.Printf("platform_required=%t\n", scope.PlatformRequired)
	fmt.Printf("container_required=%t\n", scope.ContainerRequired)
	fmt.Printf("native_required=%t\n", scope.NativeRequired)
}

type changeScope struct {
	DocsOnly          bool
	QualityRequired   bool
	PlatformRequired  bool
	ContainerRequired bool
	NativeRequired    bool
}

// classify 在缺少 push 的 before SHA 时明确选择完整验证。初始 push 没有可比较的
// 变更集，绝不能把它误判为文档改动而跳过二进制或供应链门禁。
func classify(base, head string) (changeScope, string, error) {
	if base == "" || isAllZero(base) {
		return fullScope(true), "no usable base commit; selecting full validation", nil
	}
	if head == "" {
		return changeScope{}, "", errors.New("head commit is required")
	}

	command := exec.Command("git", "diff", "--name-only", "--no-renames", "-z", base, head)
	output, err := command.Output()
	if err != nil {
		return changeScope{}, "", fmt.Errorf("diff %s..%s: %w", base, head, err)
	}
	paths := splitNULPaths(output)
	if len(paths) == 0 {
		return fullScope(true), "empty diff; selecting full validation", nil
	}
	if docsOnlyPaths(paths) {
		return changeScope{DocsOnly: true}, "only approved documentation paths changed; selecting documentation validation", nil
	}
	return fullScope(containerRelevant(paths)), "non-document change detected; selecting required validation", nil
}

func fullScope(container bool) changeScope {
	return changeScope{
		QualityRequired:   true,
		PlatformRequired:  true,
		ContainerRequired: container,
	}
}

func containerRelevant(paths []string) bool {
	for _, path := range paths {
		switch {
		case path == "Dockerfile", path == ".dockerignore", path == "go.mod", path == "go.sum", path == "LICENSE":
			return true
		case strings.HasPrefix(path, "cmd/"),
			strings.HasPrefix(path, "internal/"),
			strings.HasPrefix(path, "sdk/"),
			strings.HasPrefix(path, "ci/"),
			strings.HasPrefix(path, "tools/platformmatrix/"),
			path == "scripts/build-release.sh":
			return true
		}
	}
	return false
}

func isAllZero(value string) bool {
	return strings.Trim(value, "0") == ""
}

func splitNULPaths(value []byte) []string {
	if len(value) == 0 {
		return nil
	}
	parts := strings.Split(string(value), "\x00")
	return parts[:len(parts)-1]
}

// docsOnlyPaths 是刻意严格的 allowlist。任何代码、依赖、脚本、workflow 或未列出的
// 文件都会保留完整 CI，避免路径分类成为绕过发布验证的途径。
func docsOnlyPaths(paths []string) bool {
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		if !isApprovedDocumentationPath(path) {
			return false
		}
	}
	return true
}

func writeOutput(path string, scope changeScope) error {
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(file,
		"docs_only=%t\nquality_required=%t\nplatform_required=%t\ncontainer_required=%t\nnative_required=%t\n",
		scope.DocsOnly,
		scope.QualityRequired,
		scope.PlatformRequired,
		scope.ContainerRequired,
		scope.NativeRequired,
	); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
