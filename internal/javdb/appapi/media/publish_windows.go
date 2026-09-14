//go:build windows

package media

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// linkNoReplace 在 Windows 上用 MoveFileEx(MOVEFILE_REPLACE_EXISTING 关闭)
// 实现原子 no-replace 发布(计划 #28)。
func linkNoReplace(sourcePath, targetPath string) error {
	from, err := windows.UTF16PtrFromString(sourcePath)
	if err != nil {
		return fmt.Errorf("resolve temp path: %w", err)
	}
	to, err := windows.UTF16PtrFromString(targetPath)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}
	// 不带 MOVEFILE_REPLACE_EXISTING:目标已存在时失败,永不覆盖。
	return windows.MoveFileEx(from, to, 0)
}
