package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Load 读取认证文件；文件不存在时返回空 store。
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Store{}, nil
		}
		return nil, err
	}
	var store Store
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return &store, nil
}

// Save 以原有的临时文件 + rename 方式保存 0600 认证文件。
func Save(path string, store *Store) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	temporaryPath := path + ".tmp"
	if err := os.WriteFile(temporaryPath, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporaryPath, path)
}

// FileStore 将文件路径与互斥提交操作绑定，供 CLI 使用。
type FileStore struct {
	Path string
	mu   sync.Mutex
}

// Open 加载指定路径的 store；不存在时返回空 store 与已绑定的 FileStore。
func Open(path string) (*FileStore, *Store, error) {
	store, err := Load(path)
	if err != nil {
		return nil, nil, err
	}
	return &FileStore{Path: path}, store, nil
}

// Commit 将 store 保存到绑定路径。
func (f *FileStore) Commit(store *Store) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return Save(f.Path, store)
}
