//go:build !windows

package atomicfile

import "os"

// LinkNoReplace 在同一文件系统内以硬链接原子创建目标。
// 目标已存在时由操作系统返回 EEXIST，调用方不会覆盖已有文件。
func LinkNoReplace(sourcePath, targetPath string) error {
	return os.Link(sourcePath, targetPath)
}
