// Package releaseversion 统一拥有仓库级稳定发布版本 identity。
package releaseversion

import (
	"fmt"
	"regexp"
	"strings"
)

var stableTagPattern = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

// ParseStableTag 校验稳定发布 tag，并返回去掉前导 v 的 version。
func ParseStableTag(tag string) (string, error) {
	if !stableTagPattern.MatchString(tag) {
		return "", fmt.Errorf("release tag must be stable SemVer vX.Y.Z: %q", tag)
	}
	return strings.TrimPrefix(tag, "v"), nil
}
