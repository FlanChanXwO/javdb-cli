package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/provider"
)

func sampleResponse() *provider.Response {
	return &provider.Response{
		Source: "builtin",
		Candidates: []provider.Candidate{{
			VideoCode:  "SSIS-589",
			Similarity: 95.2,
			Frames: []provider.Frame{{
				ImageName:    "SSIS-589_01-04-53.jpg",
				Similarity:   95.2,
				Timestamp:    "01:04:53",
				ThumbnailURL: "https://avscan.cc/thumb/SSIS-589/SSIS-589_01-04-53.webp",
			}},
		}},
	}
}

func TestPutGetRoundTrip(t *testing.T) {
	store := New(t.TempDir(), 0)
	if err := store.Put("builtin", stringsRepeatHex("a"), sampleResponse()); err != nil {
		t.Fatalf("Put: %v", err)
	}
	response, ok, err := store.Get("builtin", stringsRepeatHex("a"))
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if response.Candidates[0].VideoCode != "SSIS-589" {
		t.Errorf("video code = %q", response.Candidates[0].VideoCode)
	}
	if response.Candidates[0].Frames[0].Timestamp != "01:04:53" {
		t.Errorf("frame timestamp lost: %+v", response.Candidates[0].Frames[0])
	}
}

func TestGetMissesOnMissingSource(t *testing.T) {
	store := New(t.TempDir(), 0)
	_, ok, err := store.Get("builtin", stringsRepeatHex("b"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Fatal("Get reported a hit for a missing key")
	}
}

func TestGetIsolatedBySource(t *testing.T) {
	store := New(t.TempDir(), 0)
	if err := store.Put("builtin", stringsRepeatHex("c"), sampleResponse()); err != nil {
		t.Fatal(err)
	}
	_, ok, err := store.Get("custom", stringsRepeatHex("c"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Fatal("cache leaked across sources")
	}
}

func TestTTLExpiryIsNormalMiss(t *testing.T) {
	// TTL 窗口要远大于调度抖动：fresh 命中与过期 miss 都依赖真实时钟。
	store := New(t.TempDir(), 500*time.Millisecond)
	if err := store.Put("builtin", stringsRepeatHex("d"), sampleResponse()); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := store.Get("builtin", stringsRepeatHex("d")); err != nil || !ok {
		t.Fatalf("fresh entry should hit: ok=%v err=%v", ok, err)
	}
	time.Sleep(700 * time.Millisecond)
	_, ok, err := store.Get("builtin", stringsRepeatHex("d"))
	if err != nil {
		t.Fatalf("expired entry must be a normal miss, got error: %v", err)
	}
	if ok {
		t.Fatal("expired entry reported as hit")
	}
}

func TestCorruptedCacheErrorsExplicitly(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "builtin.json")
	if err := os.WriteFile(file, []byte("{corrupted"), 0o600); err != nil {
		t.Fatal(err)
	}
	store := New(dir, 0)
	if _, _, err := store.Get("builtin", stringsRepeatHex("e")); err == nil {
		t.Fatal("Get must report a corrupted cache, not a miss")
	}
	if err := store.Put("builtin", stringsRepeatHex("e"), sampleResponse()); err == nil {
		t.Fatal("Put must not silently overwrite a corrupted cache")
	}
}

func TestStatsCountsPerSource(t *testing.T) {
	store := New(t.TempDir(), 0)
	for _, source := range []string{"builtin", "custom"} {
		for index := 0; index < 3; index++ {
			if err := store.Put(source, stringsRepeatHex(fmt.Sprintf("%d", index)), sampleResponse()); err != nil {
				t.Fatal(err)
			}
		}
	}
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats["builtin"] != 3 || stats["custom"] != 3 {
		t.Errorf("stats = %+v", stats)
	}
	if sorted := SortedSources(stats); len(sorted) != 2 || sorted[0] != "builtin" || sorted[1] != "custom" {
		t.Errorf("sorted sources = %v", sorted)
	}
}

func TestClearBySourceAndAll(t *testing.T) {
	store := New(t.TempDir(), 0)
	if err := store.Put("builtin", stringsRepeatHex("f"), sampleResponse()); err != nil {
		t.Fatal(err)
	}
	if err := store.Put("custom", stringsRepeatHex("f"), sampleResponse()); err != nil {
		t.Fatal(err)
	}
	if err := store.Clear("builtin"); err != nil {
		t.Fatalf("Clear builtin: %v", err)
	}
	if _, ok, err := store.Get("builtin", stringsRepeatHex("f")); err != nil || ok {
		t.Fatalf("builtin cache should be gone: ok=%v err=%v", ok, err)
	}
	if _, ok, err := store.Get("custom", stringsRepeatHex("f")); err != nil || !ok {
		t.Fatalf("custom cache must survive source-scoped clear: ok=%v err=%v", ok, err)
	}
	if err := store.Clear(""); err != nil {
		t.Fatalf("Clear all: %v", err)
	}
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if len(stats) != 0 {
		t.Errorf("stats after clear-all = %+v", stats)
	}
}

func TestCacheFilesArePrivateAndAtomic(t *testing.T) {
	dir := t.TempDir()
	store := New(dir, 0)
	if err := store.Put("builtin", stringsRepeatHex("g"), sampleResponse()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "builtin.json"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("cache file permissions = %o, want 0600", info.Mode().Perm())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "builtin.json" {
			t.Errorf("stray file in cache dir: %s", entry.Name())
		}
	}
}

func TestConcurrentPutAndGet(t *testing.T) {
	store := New(t.TempDir(), 0)
	var wait sync.WaitGroup
	for index := 0; index < 16; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			key := stringsRepeatHex(fmt.Sprintf("%02d", index))
			if err := store.Put("builtin", key, sampleResponse()); err != nil {
				t.Errorf("Put: %v", err)
			}
			if _, ok, err := store.Get("builtin", key); err != nil || !ok {
				t.Errorf("Get: ok=%v err=%v", ok, err)
			}
		}(index)
	}
	wait.Wait()
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats["builtin"] != 16 {
		t.Errorf("stats = %+v, want 16 entries", stats)
	}
}

func TestSanitizeSourceName(t *testing.T) {
	if got := sanitizeSourceName("my source/1"); got != "my_source_1" {
		t.Errorf("sanitized = %q", got)
	}
	if got := sanitizeSourceName("///"); got != "source" {
		t.Errorf("empty source sanitized = %q", got)
	}
}

func stringsRepeatHex(value string) string {
	// 生成 64 位小写 hex 样式的 key。
	result := value
	for len(result) < 64 {
		result += "0"
	}
	return result
}

// TestNullResponseEntryIsExplicitError response: null 的条目必须显式报错，
// 不得当作命中（命中后解引用会 panic）。
func TestNullResponseEntryIsExplicitError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "builtin.json")
	if err := os.WriteFile(file, []byte(`{"k":{"written_at":"2026-08-13T00:00:00Z","response":null}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := New(dir, 0)
	if _, ok, err := store.Get("builtin", "k"); err == nil || ok {
		t.Fatalf("null response entry must be an explicit error: ok=%v err=%v", ok, err)
	}
	if err := store.Put("builtin", "k", nil); err == nil {
		t.Fatal("Put must reject a nil response")
	}
}

// TestMissingWrittenAtIsExplicitError 缺少 written_at 的合法 JSON 条目必须
// 显式报错，不得被静默当作过期 miss。
func TestMissingWrittenAtIsExplicitError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "builtin.json")
	if err := os.WriteFile(file, []byte(`{"k":{"response":{"source":"builtin"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	store := New(dir, 0)
	if _, ok, err := store.Get("builtin", "k"); err == nil || ok {
		t.Fatalf("entry without written_at must be an explicit error: ok=%v err=%v", ok, err)
	}
}

// TestPutRejectsSemanticallyCorruptExistingCache Put 遇到语义损坏的既有条目
// 必须显式报错，不静默保留或覆盖。
func TestPutRejectsSemanticallyCorruptExistingCache(t *testing.T) {
	for _, content := range []string{
		`{"k":{"written_at":"2026-08-13T00:00:00Z","response":null}}`,
		`{"k":{"response":{"source":"builtin"}}}`,
	} {
		dir := t.TempDir()
		file := filepath.Join(dir, "builtin.json")
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		store := New(dir, 0)
		if err := store.Put("builtin", "other", sampleResponse()); err == nil {
			t.Fatalf("Put silently accepted corrupt cache: %s", content)
		}
	}
}
