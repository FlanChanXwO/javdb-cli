// Package authstore 只负责默认认证文件的路径、目录与 store 打开。
package authstore

import (
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
)

// Open 解析默认认证路径、确保配置目录存在，再打开认证 store。
// 不吞掉路径、目录或文件错误。
func Open() (*auth.FileStore, *auth.Store, error) {
	path, err := paths.AuthPath()
	if err != nil {
		return nil, nil, err
	}
	if _, err := paths.EnsureDir(); err != nil {
		return nil, nil, err
	}
	return auth.Open(path)
}
