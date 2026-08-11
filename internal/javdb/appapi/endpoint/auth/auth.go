// Package auth 负责登录、启动信息和用户身份解析 endpoint。
package auth

import (
	"encoding/json"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/client"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/codec"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
)

// AuthEndpoint 提供登录、启动信息和用户身份解析 capability。
type AuthEndpoint struct {
	c *client.Client
}

// NewAuth 用共享 transport 构造 auth capability。
func NewAuth(c *client.Client) *AuthEndpoint {
	return &AuthEndpoint{c: c}
}

// Login 提交用户名密码并把返回 token 写入 client。
func (e *AuthEndpoint) Login(username, password string) (string, error) {
	var data map[string]json.RawMessage
	if err := e.c.PostFormJSON("/api/v1/sessions", map[string]string{
		"username": username,
		"password": password,
	}, &data); err != nil {
		return "", err
	}
	token := codec.RawString(data, "token")
	if token == "" {
		token = codec.RawString(data, "access_token")
	}
	if token == "" {
		return "", &model.Error{Action: "NoToken", Message: "login response had no token"}
	}
	e.c.SetToken(token)
	return token, nil
}

// Startup 返回 GET /api/v1/startup 的 data。
func (e *AuthEndpoint) Startup() (map[string]json.RawMessage, error) {
	var data map[string]json.RawMessage
	if err := e.c.GetJSON("/api/v1/startup", nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// Users 返回 GET /api/v1/users 的 profile data。
func (e *AuthEndpoint) Users() (map[string]json.RawMessage, error) {
	var data map[string]json.RawMessage
	if err := e.c.GetJSON("/api/v1/users", nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// ResolveUserID 依次尝试 JWT claims、/users 和 /startup.user。
func (e *AuthEndpoint) ResolveUserID(token string) (int64, string, error) {
	if token == "" {
		token = e.c.Token()
	}
	if id, name, ok := codec.UserIDFromJWT(token); ok {
		return id, name, nil
	}
	if data, err := e.Users(); err == nil {
		if id, name, ok := codec.UserIDFromMap(data); ok {
			return id, name, nil
		}
		if raw, ok := data["user"]; ok {
			var nested map[string]json.RawMessage
			if json.Unmarshal(raw, &nested) == nil {
				if id, name, ok := codec.UserIDFromMap(nested); ok {
					return id, name, nil
				}
			}
		}
	}
	if data, err := e.Startup(); err == nil {
		if raw, ok := data["user"]; ok {
			var nested map[string]json.RawMessage
			if json.Unmarshal(raw, &nested) == nil {
				if id, name, ok := codec.UserIDFromMap(nested); ok {
					return id, name, nil
				}
			}
		}
	}
	return 0, "", &model.Error{Action: "NoUserID", Message: "could not resolve numeric user id from token/users/startup"}
}
