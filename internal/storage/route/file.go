// Package route 管理自动选线路由缓存（~/.javdb-cli/route.json）的原子读写。
//
// schema 只保存 {"host":"https://..."}，不保存 proxy、token、时间戳、测速历史或 fallback
// 状态。缺失文件是"无缓存"；损坏、未知字段、空 host 或非法 URL 都显式报错，不伪装为 miss。
package route

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ErrEmptyHost 表示缓存 host 为空。
var ErrEmptyHost = errors.New("route cache host is empty")

// Document 是 route.json 的磁盘 schema。
type Document struct {
	Host string `json:"host"`
}

// Load 读取 route cache。缺失文件返回 ok=false、无错误；损坏 JSON、未知字段、空 host 或
// 非法 URL 返回显式错误。
func Load(path string) (Document, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Document{}, false, nil
		}
		return Document{}, false, fmt.Errorf("route cache: read %s: %w", path, err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var doc Document
	if err := decoder.Decode(&doc); err != nil {
		return Document{}, false, fmt.Errorf("route cache: decode %s: %w", path, err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Document{}, false, fmt.Errorf("route cache: trailing data in %s", path)
	}
	if err := ValidateHost(doc.Host); err != nil {
		return Document{}, false, fmt.Errorf("route cache: %s: %w", path, err)
	}
	return doc, true, nil
}

// Save 原子写入 route cache：同目录私有临时文件、完整写入、Sync、rename 替换，目标 0600。
// 任一步失败都清理临时文件并显式返回错误，不留下半成品。
func Save(path string, doc Document) error {
	if err := ValidateHost(doc.Host); err != nil {
		return fmt.Errorf("route cache: %w", err)
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("route cache: mkdir %s: %w", directory, err)
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("route cache: marshal: %w", err)
	}
	file, err := os.CreateTemp(directory, ".route-*.tmp")
	if err != nil {
		return fmt.Errorf("route cache: create temp in %s: %w", directory, err)
	}
	temporaryPath := file.Name()
	complete := false
	defer func() {
		if complete {
			return
		}
		_ = file.Close()
		_ = os.Remove(temporaryPath)
	}()
	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("route cache: chmod temp: %w", err)
	}
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("route cache: write temp: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("route cache: sync temp: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("route cache: close temp: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("route cache: replace %s: %w", path, err)
	}
	complete = true
	return nil
}

// ValidateHost 校验 host 为不含 query/fragment 的绝对 http/https URL。
func ValidateHost(host string) error {
	if strings.TrimSpace(host) == "" {
		return ErrEmptyHost
	}
	u, err := url.Parse(host)
	if err != nil || !u.IsAbs() || u.Host == "" {
		return fmt.Errorf("route cache host is not a valid URL: %q", host)
	}
	if !strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https") {
		return fmt.Errorf("route cache host has unsupported scheme: %q", host)
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("route cache host must not contain query/fragment: %q", host)
	}
	return nil
}
