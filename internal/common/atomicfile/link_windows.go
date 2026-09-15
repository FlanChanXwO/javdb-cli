//go:build windows

package atomicfile

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// LinkNoReplace 在 Windows 上以 CreateHardLink 原子创建目标，保持与 Unix
// os.Link 相同的 no-replace 语义。
func LinkNoReplace(sourcePath, targetPath string) error {
	newLink, err := windows.UTF16PtrFromString(targetPath)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}
	existingFile, err := windows.UTF16PtrFromString(sourcePath)
	if err != nil {
		return fmt.Errorf("resolve source path: %w", err)
	}
	if err := windows.CreateHardLink(newLink, existingFile, 0); err != nil {
		if err == windows.ERROR_FILE_EXISTS || err == windows.ERROR_ALREADY_EXISTS {
			return os.ErrExist
		}
		return fmt.Errorf("create target link: %w", err)
	}
	return nil
}
