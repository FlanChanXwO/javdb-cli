// Package cache 实现反搜响应的文件缓存：key 为 source + 原图 SHA-256，
// 只保存规范化响应与写入时间（不保存图片、鉴权 header 或 JavDB 详情），
// 原子写入 0600，TTL 默认 30 天。
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/provider"
)

// DefaultTTL 是缓存条目默认有效期（30 天）。
const DefaultTTL = 30 * 24 * time.Hour

// Store 是目录化的反搜响应缓存；每个 source 一个 JSON 文件。
type Store struct {
	dir string
	ttl time.Duration
	mu  sync.Mutex
}

// New 创建缓存 store；ttl <= 0 时使用 DefaultTTL。
func New(dir string, ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &Store{dir: dir, ttl: ttl}
}

// entry 是单个缓存条目；不包含任何敏感字段。
type entry struct {
	WrittenAt time.Time          `json:"written_at"`
	Response  *provider.Response `json:"response"`
}

func (s *Store) fileFor(source string) string {
	return filepath.Join(s.dir, sanitizeSourceName(source)+".json")
}

// sanitizeSourceName 把 source 名约束为安全文件名片段。
func sanitizeSourceName(source string) string {
	var builder strings.Builder
	for _, character := range source {
		switch {
		case character >= 'a' && character <= 'z',
			character >= 'A' && character <= 'Z',
			character >= '0' && character <= '9',
			character == '-', character == '_':
			builder.WriteRune(character)
		default:
			builder.WriteRune('_')
		}
	}
	name := builder.String()
	if strings.Trim(name, "_") == "" {
		return "source"
	}
	return name
}

// Get 读取 source 的 key 缓存；过期条目是正常 miss（惰性清理），损坏或读取
// 失败显式报错，绝不伪装成 miss。
func (s *Store) Get(source, key string) (*provider.Response, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	file := s.fileFor(source)
	entries, err := s.readEntries(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		return nil, false, err
	}
	item, exists := entries[key]
	if !exists {
		return nil, false, nil
	}
	// response 为 null 或 written_at 缺失（零时间）的条目是损坏缓存：
	// 显式报错，绝不当作命中（命中后下游解引用会 panic；零时间被静默当
	// 过期删除会掩盖损坏）。
	if item.Response == nil {
		return nil, false, fmt.Errorf("reverse search cache %q contains a null response", file)
	}
	if item.WrittenAt.IsZero() {
		return nil, false, fmt.Errorf("reverse search cache %q contains an entry without written_at", file)
	}
	if time.Since(item.WrittenAt) > s.ttl {
		delete(entries, key)
		// 过期清理是 best-effort：重写失败不把正常 miss 变成错误。
		_ = s.writeEntries(file, entries)
		return nil, false, nil
	}
	return item.Response, true, nil
}

// Put 写入 source 的 key 缓存；已有损坏缓存显式报错，不静默覆盖。
func (s *Store) Put(source, key string, response *provider.Response) error {
	if response == nil {
		return fmt.Errorf("refuse to cache a nil reverse search response")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	file := s.fileFor(source)
	entries, err := s.readEntries(file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	// 既有条目语义损坏（null response 或缺 written_at）必须显式报错，
	// 不静默保留或覆盖。
	for existingKey, existing := range entries {
		if existing.Response == nil {
			return fmt.Errorf("reverse search cache %q contains a null response for key %q", file, existingKey)
		}
		if existing.WrittenAt.IsZero() {
			return fmt.Errorf("reverse search cache %q contains an entry without written_at for key %q", file, existingKey)
		}
	}
	if entries == nil {
		entries = map[string]entry{}
	}
	entries[key] = entry{WrittenAt: time.Now().UTC(), Response: response}
	return s.writeEntries(file, entries)
}

// Stats 返回每个 source 的条目数（按名称排序，便于稳定输出）。
func (s *Store) Stats() (map[string]int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	files, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]int{}, nil
		}
		return nil, err
	}
	stats := map[string]int{}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		entries, err := s.readEntries(filepath.Join(s.dir, file.Name()))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		stats[strings.TrimSuffix(file.Name(), ".json")] = len(entries)
	}
	return stats, nil
}

// Clear 清空指定 source 的缓存；source 为空时清空全部反搜缓存。
func (s *Store) Clear(source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if source == "" {
		if err := os.RemoveAll(s.dir); err != nil {
			return fmt.Errorf("clear reverse search cache: %w", err)
		}
		return nil
	}
	file := s.fileFor(source)
	if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear reverse search cache for %q: %w", source, err)
	}
	return nil
}

func (s *Store) readEntries(file string) (map[string]entry, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var entries map[string]entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("reverse search cache %q is corrupted: %w", file, err)
	}
	return entries, nil
}

// writeEntries 以原子 0600 写入缓存文件。
func (s *Store) writeEntries(file string, entries map[string]entry) error {
	data, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("encode reverse search cache: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(file), ".reverse-search-cache-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			_ = temporary.Close()
		}
		_ = os.Remove(temporaryPath)
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
	if err := os.Rename(temporaryPath, file); err != nil {
		return err
	}
	return nil
}

// SortedSources 返回统计中的 source 名列表（升序）。
func SortedSources(stats map[string]int) []string {
	names := make([]string, 0, len(stats))
	for name := range stats {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
