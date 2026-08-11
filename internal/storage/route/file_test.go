package route

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestLoadMissingCacheReturnsNoCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "route.json")
	doc, ok, err := Load(path)
	if err != nil {
		t.Fatalf("Load(missing) error = %v", err)
	}
	if ok || doc.Host != "" {
		t.Fatalf("Load(missing) = %+v, %v; want empty, false", doc, ok)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "route.json")
	const host = "https://apidd.spthgb.com"
	if err := Save(path, Document{Host: host}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	doc, ok, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !ok || doc.Host != host {
		t.Fatalf("Load() = %+v, %v; want host %s", doc, ok, host)
	}
}

func TestSaveSetsPrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions are not meaningful on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "route.json")
	if err := Save(path, Document{Host: "https://x.example"}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("route.json permissions = %04o, want 0600", got)
	}
	if got := info.Mode().Perm() & 0o077; got != 0 {
		t.Fatalf("route.json has group/other bits: %04o", info.Mode().Perm())
	}
}

func TestSaveReplacesExistingCache(t *testing.T) {
	path := filepath.Join(t.TempDir(), "route.json")
	if err := Save(path, Document{Host: "https://first.example"}); err != nil {
		t.Fatalf("first Save() error = %v", err)
	}
	if err := Save(path, Document{Host: "https://second.example"}); err != nil {
		t.Fatalf("second Save() error = %v", err)
	}
	doc, ok, err := Load(path)
	if err != nil || !ok || doc.Host != "https://second.example" {
		t.Fatalf("Load() after replace = %+v, %v, %v", doc, ok, err)
	}
}

func TestLoadRejectsCorruptAndStrict(t *testing.T) {
	cases := []struct {
		name string
		data string
	}{
		{"invalid json", `{not json`},
		{"unknown field", `{"host":"https://x.example","proxy":"http://p"}`},
		{"trailing data", `{"host":"https://x.example"}{"host":"https://y.example"}`},
		{"empty host", `{"host":""}`},
		{"empty object", `{}`},
		{"host not string", `{"host":123}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "route.json")
			if err := os.WriteFile(path, []byte(tc.data), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			if _, ok, err := Load(path); err == nil || ok {
				t.Fatalf("Load(%q) = ok=%v, err=%v; want error", tc.data, ok, err)
			}
		})
	}
}

func TestLoadRejectsInvalidHostURL(t *testing.T) {
	for _, host := range []string{
		"not-a-url",
		"ftp://x.example",
		"https://",
		"/relative/path",
		"https://x.example?a=1",
		"https://x.example#frag",
		"   ",
	} {
		t.Run(host, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "route.json")
			if err := os.WriteFile(path, []byte(fmt.Sprintf(`{"host":%q}`, host)), 0o600); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}
			if _, ok, err := Load(path); err == nil || ok {
				t.Fatalf("Load(host %q) = ok=%v, err=%v; want error", host, ok, err)
			}
		})
	}
}

func TestSaveRejectsInvalidHostBeforeWriting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "route.json")
	if err := Save(path, Document{Host: "not-a-url"}); err == nil {
		t.Fatal("Save(invalid host) unexpectedly succeeded")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Save(invalid host) wrote a file: %v", err)
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, ".route-*.tmp")); len(matches) != 0 {
		t.Fatalf("Save(invalid host) left temp files: %v", matches)
	}
}

func TestSaveFailureCleansUpTemp(t *testing.T) {
	dir := t.TempDir()
	// 把目标路径变成一个非空目录，让 rename 替换失败。
	target := filepath.Join(dir, "route.json")
	if err := os.MkdirAll(filepath.Join(target, "sub"), 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := Save(target, Document{Host: "https://x.example"}); err == nil {
		t.Fatal("Save(replace over non-empty dir) unexpectedly succeeded")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".route-*.tmp"))
	if len(matches) != 0 {
		t.Fatalf("Save failure left temp files: %v", matches)
	}
}

func TestConcurrentSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "route.json")
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			host := fmt.Sprintf("https://host-%d.example", i)
			if err := Save(path, Document{Host: host}); err != nil {
				t.Errorf("Save error = %v", err)
				return
			}
			if _, ok, err := Load(path); err != nil || !ok {
				t.Errorf("Load error = %v ok=%v", err, ok)
			}
		}(i)
	}
	group.Wait()
	doc, ok, err := Load(path)
	if err != nil || !ok {
		t.Fatalf("final Load error = %v ok=%v", err, ok)
	}
	if !strings.HasPrefix(doc.Host, "https://host-") {
		t.Fatalf("final host = %q, want one of the concurrent writers", doc.Host)
	}
}

func TestLoadReadErrorIsExplicit(t *testing.T) {
	// 目录不可读时 ReadFile 报错，应显式返回而非伪装为 miss。
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions are not meaningful on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root; read-only dir does not restrict")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o000); err != nil {
		t.Fatalf("Chmod() error = %v", err)
	}
	defer os.Chmod(dir, 0o700) //nolint:errcheck
	path := filepath.Join(dir, "route.json")
	if _, ok, err := Load(path); err == nil || ok {
		t.Fatalf("Load(unreadable dir) = ok=%v, err=%v; want error", ok, err)
	}
}
