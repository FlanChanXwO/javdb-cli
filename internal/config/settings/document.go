package settings

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Document 是 config.toml 的结构化树视图：保存时保留未修改的表、source 数组
// 与未知键的语义值，不再整体重编码 typed struct。
type Document struct {
	tree map[string]any
}

// LoadDocument 读取配置树；文件缺失返回空文档。
func LoadDocument(path string) (*Document, error) {
	document := &Document{tree: map[string]any{}}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return document, nil
		}
		return nil, err
	}
	if err := toml.Unmarshal(data, &document.tree); err != nil {
		return nil, err
	}
	return document, nil
}

// Set 通过点路径写入标量值，必要时创建中间表。
func (d *Document) Set(key string, value any) error {
	if d == nil {
		return errors.New("settings document is nil")
	}
	parts := strings.Split(key, ".")
	if len(parts) == 0 || parts[0] == "" {
		return fmt.Errorf("invalid config key %q", key)
	}
	current := d.tree
	for _, part := range parts[:len(parts)-1] {
		next, exists := current[part]
		if !exists {
			table := map[string]any{}
			current[part] = table
			current = table
			continue
		}
		table, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("config key %q traverses a non-table value", key)
		}
		current = table
	}
	current[parts[len(parts)-1]] = value
	return nil
}

// Delete 通过点路径删除键；键不存在是 no-op。
func (d *Document) Delete(key string) error {
	if d == nil {
		return errors.New("settings document is nil")
	}
	parts := strings.Split(key, ".")
	if len(parts) == 0 || parts[0] == "" {
		return fmt.Errorf("invalid config key %q", key)
	}
	current := d.tree
	for _, part := range parts[:len(parts)-1] {
		next, exists := current[part]
		if !exists {
			return nil
		}
		table, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("config key %q traverses a non-table value", key)
		}
		current = table
	}
	delete(current, parts[len(parts)-1])
	return nil
}

// Save 把整棵树以原子方式写回 path（0600）。
func (d *Document) Save(path string) error {
	if d == nil {
		return errors.New("settings document is nil")
	}
	data, err := toml.Marshal(d.tree)
	if err != nil {
		return fmt.Errorf("encode settings document: %w", err)
	}
	return writeFileAtomic(path, data)
}

// writeFileAtomic 以同目录临时文件 + rename 原子写 0600 文件。
func writeFileAtomic(path string, data []byte) (resultErr error) {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			resultErr = errors.Join(resultErr, temporary.Close())
		}
		if removeErr := os.Remove(temporaryPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) && resultErr == nil {
			resultErr = removeErr
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		closed = true
		return err
	}
	closed = true
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return nil
}
