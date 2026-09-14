//go:build !windows

package media

import "os"

// linkNoReplace 用 os.Link 实现原子 no-replace 发布(计划 #28):
// 目标已存在时 link 返回 EEXIST,永不覆盖已有文件;
// 两个进程同时写相同 target 时一方成功另一方 ErrExist。
func linkNoReplace(sourcePath, targetPath string) error {
	return os.Link(sourcePath, targetPath)
}
