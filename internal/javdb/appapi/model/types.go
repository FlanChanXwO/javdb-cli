// Package model 定义 JavDB App API 的稳定配置、响应和领域类型。
package model

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// AppVersion 是请求使用的 App 版本字符串。
	AppVersion = "1.9.28"
	// AppVersionNumber 是请求使用的数值版 App 版本。
	AppVersionNumber = "10928"
	// UserAgent 是 App API 使用的 User-Agent。
	UserAgent = "Dart/3.4 (dart:io)"

	// HostMirror 是默认的 JavDB API 镜像地址。
	HostMirror = "https://jdforrepam.com"
	// HostMain 是 JavDB 主站地址。
	HostMain = "https://javdb.com"
)

// Zones 是电影类型到 App API 数字掩码的映射。
var Zones = map[string]int{
	"censored":   0,
	"uncensored": 1,
	"western":    2,
	"fc2":        3,
}

// MainFlags 是 filter_by 支持的 main 属性字母。
var MainFlags = map[string]bool{
	"p": true, "m": true, "c": true, "s": true, "i": true, "v": true,
}

// EntityLetters 将实体类型映射为 filter_by 字母。
var EntityLetters = map[string]string{
	"actor":    "a",
	"series":   "s",
	"maker":    "m",
	"director": "d",
	"code":     "c",
	"list":     "l",
}

// Options 配置一个签名的 App API client。
type Options struct {
	Host       string
	Token      string
	DeviceUUID string
	Proxy      string
	Timeout    time.Duration
	Lang       string
	// Device profile（可选覆盖项）。
	AppChannel    string
	SystemVersion string
	DeviceModel   string
	DeviceName    string
}

// SearchOptions 配置 /api/v2/search。
//
// Zone 支持 censored、uncensored、western、fc2，以及省略 movie_type 的 all/空值。
type SearchOptions struct {
	Page     int
	Limit    int
	Zone     string
	Sort     string
	FilterBy string
	Type     string
}

// BrowseOptions 配置 /api/v1/movies/tags 分类浏览。
type BrowseOptions struct {
	Zone   string
	Main   []string
	TagIDs []string
	Year   string
	Month  string
	Sort   string
	Order  string
	Page   int
	Limit  int
}

// EntityMoviesOptions 配置实体作品列表。
type EntityMoviesOptions struct {
	Zone  string
	Page  int
	Limit int
	Sort  string
	Order string
	Main  []string
	Tags  []string
}

// LoginResponse 是 POST /api/v1/sessions 的 data payload。
type LoginResponse struct {
	Token       string                     `json:"token"`
	AccessToken string                     `json:"access_token"`
	User        json.RawMessage            `json:"user"`
	Raw         map[string]json.RawMessage `json:"-"`
}

// Error 表示 App API 返回的服务端错误（success:0）。
type Error struct {
	Action  string
	Message string
}

// Error 实现 error。
func (e *Error) Error() string {
	if e.Action == "" && e.Message == "" {
		return "api error"
	}
	return fmt.Sprintf("%s: %s", e.Action, e.Message)
}

// AuthRequired 表示 bearer token 无效或已过期，需要重新登录。
type AuthRequired struct {
	API Error
}

// Error 实现 error。
func (e *AuthRequired) Error() string { return e.API.Error() }

// Unwrap 暴露底层的 API Error，保持 errors.Is/As 兼容。
func (e *AuthRequired) Unwrap() error { return &e.API }
