//go:build windows

package media

import (
	"fmt"

	"golang.org/x/sys/windows"
)

// linkNoReplace 在 Windows 上用 CreateHardLink 实现与 Unix os.Link 相同的
// 原子 no-replace 语义(计划 #28):目标已存在时 link 返回错误,永不覆盖;
// 两个进程同时写相同 target 时一方成功另一方 ErrExist。
// 不用 MoveFileEx:它会移除临时文件,Unix 路径发布后还需删除 tmp,
// 行为不一致会导致 Windows 上 remove 已不存在的文件报错。
func linkNoReplace(sourcePath, targetPath string) error {
	from, err := windows.UTF16PtrFromString(targetPath)
	if err != nil {
		return fmt.Errorf("resolve target path: %w", err)
	}
	to, err := windows.UTF16PtrFromString(sourcePath)
	if err != nil {
		return fmt.Errorf("resolve temp path: %w", err)
	}
	// CreateHardLink(newLink, existingFileName, reserved):
	// newLink 是要创建的链接,targetPath;existingFileName 是已存在的源,tmp。
	if err := windows.CreateHardLink(from, to, 0); err != nil {
		// 目标已存在时映射为 ErrExist,与 Unix os.Link 一致。
		return fmt.Errorf("publish media file: %w", err)
	}
	return nil
}
