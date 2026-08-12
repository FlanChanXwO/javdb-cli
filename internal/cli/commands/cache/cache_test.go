package cache

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/cache"
	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/provider"
)

func executeCache(t *testing.T, home string, args ...string) (string, string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	streams := invocation.NewStreams(strings.NewReader(""), &out, &errb)
	command := New(streams)
	command.SetArgs(args)
	err := command.Execute()
	return out.String(), errb.String(), err
}

func seedCache(t *testing.T, home string) {
	t.Helper()
	dir, err := paths.ReverseSearchCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	store := cache.New(dir, 0)
	response := &provider.Response{Source: "builtin", Candidates: []provider.Candidate{{VideoCode: "SSIS-589"}}}
	if err := store.Put("builtin", strings.Repeat("a", 64), response); err != nil {
		t.Fatal(err)
	}
	if err := store.Put("custom", strings.Repeat("b", 64), response); err != nil {
		t.Fatal(err)
	}
}

func isolateCacheHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func TestCacheReverseSearchShowsStats(t *testing.T) {
	home := isolateCacheHome(t)
	seedCache(t, home)
	out, _, err := executeCache(t, home, "reverse-search")
	if err != nil {
		t.Fatalf("reverse-search: %v", err)
	}
	if !strings.Contains(out, "builtin=1") || !strings.Contains(out, "custom=1") {
		t.Errorf("stats output = %q", out)
	}
}

func TestCacheReverseSearchEmpty(t *testing.T) {
	home := isolateCacheHome(t)
	out, _, err := executeCache(t, home, "reverse-search")
	if err != nil {
		t.Fatalf("reverse-search: %v", err)
	}
	if !strings.Contains(out, "empty") {
		t.Errorf("empty cache output = %q", out)
	}
}

func TestCacheReverseSearchBySource(t *testing.T) {
	home := isolateCacheHome(t)
	seedCache(t, home)
	out, _, err := executeCache(t, home, "reverse-search", "--source", "builtin")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out) != "builtin=1" {
		t.Errorf("source-scoped stats = %q", out)
	}
}

func TestCacheReverseSearchClearScoped(t *testing.T) {
	home := isolateCacheHome(t)
	seedCache(t, home)
	out, _, err := executeCache(t, home, "reverse-search", "--clear", "--source", "builtin")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "cleared") {
		t.Errorf("clear output = %q", out)
	}
	dir, err := paths.ReverseSearchCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "builtin.json")); !os.IsNotExist(err) {
		t.Error("builtin cache file still exists after scoped clear")
	}
	if _, err := os.Stat(filepath.Join(dir, "custom.json")); err != nil {
		t.Error("custom cache file removed by scoped clear")
	}
}

func TestCacheReverseSearchClearAll(t *testing.T) {
	home := isolateCacheHome(t)
	seedCache(t, home)
	out, _, err := executeCache(t, home, "reverse-search", "--clear")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "all") {
		t.Errorf("clear-all output = %q", out)
	}
	dir, err := paths.ReverseSearchCacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("cache directory still exists after clear-all")
	}
}
