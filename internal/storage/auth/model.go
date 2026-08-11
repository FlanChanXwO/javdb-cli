// Package auth 持久化 ~/.javdb-cli/auth.json 中的多账户认证资料。
package auth

import "errors"

// Account 表示一个已保存的登录账户；密码按现有格式明文保存。
type Account struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

// Store 是磁盘上的多账户 JSON 文档。
type Store struct {
	DefaultUserID int64     `json:"default_user_id"`
	Accounts      []Account `json:"accounts"`
}

// ErrNotFound 表示账户查询没有匹配项。
var ErrNotFound = errors.New("account not found")
