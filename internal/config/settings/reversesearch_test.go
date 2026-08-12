package settings

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadFileAppliesReverseSearchDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("host = \"main\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	rs := loaded.ReverseSearch
	if rs.DefaultSource != DefaultReverseSearchSource {
		t.Errorf("default_source = %q", rs.DefaultSource)
	}
	if !rs.CacheEnabled() {
		t.Error("cache must default to enabled")
	}
	if rs.CacheTTL != "720h" || rs.Retries != 3 || rs.RetryWait != "30s" || rs.RequestTimeout != "60s" {
		t.Errorf("unexpected defaults: %+v", rs)
	}
}

func TestResolveReverseSearchExpandsEnvHeaders(t *testing.T) {
	loaded := Defaults()
	loaded.ReverseSearch = ReverseSearchSettings{
		DefaultSource:  "custom",
		CacheTTL:       "720h",
		Retries:        3,
		RetryWait:      "30s",
		RequestTimeout: "60s",
		Sources: []ReverseSearchSource{{
			Name: "custom",
			URL:  "https://example.test/search",
			Headers: map[string]string{
				"Authorization": "Bearer ${ENV:REVERSE_SEARCH_TOKEN}",
				"X-Static":      "static-value",
			},
		}},
	}
	getenv := func(name string) string {
		if name == "REVERSE_SEARCH_TOKEN" {
			return "expanded-secret"
		}
		return ""
	}
	resolved, err := ResolveReverseSearch(loaded, getenv)
	if err != nil {
		t.Fatalf("ResolveReverseSearch: %v", err)
	}
	if got := resolved.Sources[0].Headers["Authorization"]; got != "Bearer expanded-secret" {
		t.Errorf("expanded header = %q", got)
	}
	if got := resolved.Sources[0].Headers["X-Static"]; got != "static-value" {
		t.Errorf("static header = %q", got)
	}
	if resolved.DefaultSource != "custom" || resolved.Cache != true {
		t.Errorf("resolved = %+v", resolved)
	}
	if resolved.CacheTTL != 720*time.Hour {
		t.Errorf("cache ttl = %v", resolved.CacheTTL)
	}
}

func TestResolveReverseSearchMissingEnvReportsNameOnly(t *testing.T) {
	loaded := Defaults()
	loaded.ReverseSearch = ReverseSearchSettings{
		DefaultSource: "builtin",
		Sources: []ReverseSearchSource{{
			Name: "custom",
			URL:  "https://example.test/search",
			Headers: map[string]string{
				"Authorization": "Bearer ${ENV:REVERSE_SEARCH_TOKEN}",
			},
		}},
	}
	_, err := ResolveReverseSearch(loaded, func(string) string { return "" })
	if err == nil {
		t.Fatal("ResolveReverseSearch accepted a missing env reference")
	}
	if !strings.Contains(err.Error(), "REVERSE_SEARCH_TOKEN") {
		t.Errorf("error must name the missing variable: %v", err)
	}
	if strings.Contains(err.Error(), "Bearer") {
		t.Errorf("error leaks header material: %v", err)
	}
}

func TestResolveReverseSearchRejectsInvalidSources(t *testing.T) {
	base := ReverseSearchSettings{
		DefaultSource: "builtin",
		Sources:       []ReverseSearchSource{{Name: "custom", URL: "https://example.test/search"}},
	}
	for _, tc := range []struct {
		name   string
		mutate func(*ReverseSearchSettings)
	}{
		{name: "duplicate source name", mutate: func(rs *ReverseSearchSettings) {
			rs.Sources = append(rs.Sources, ReverseSearchSource{Name: "custom", URL: "https://example.test/other"})
		}},
		{name: "reserved builtin name", mutate: func(rs *ReverseSearchSettings) {
			rs.Sources = append(rs.Sources, ReverseSearchSource{Name: "builtin", URL: "https://example.test/other"})
		}},
		{name: "empty source name", mutate: func(rs *ReverseSearchSettings) {
			rs.Sources = append(rs.Sources, ReverseSearchSource{Name: "", URL: "https://example.test/other"})
		}},
		{name: "non http url", mutate: func(rs *ReverseSearchSettings) {
			rs.Sources = append(rs.Sources, ReverseSearchSource{Name: "evil", URL: "ftp://example.test/search"})
		}},
		{name: "unsafe source name characters", mutate: func(rs *ReverseSearchSettings) {
			rs.Sources = append(rs.Sources, ReverseSearchSource{Name: "a/b", URL: "https://example.test/search"})
		}},
		{name: "unknown default source", mutate: func(rs *ReverseSearchSettings) {
			rs.DefaultSource = "ghost"
		}},
		{name: "bad cache ttl", mutate: func(rs *ReverseSearchSettings) {
			rs.CacheTTL = "not-a-duration"
		}},
		{name: "bad retry wait", mutate: func(rs *ReverseSearchSettings) {
			rs.RetryWait = "0s"
		}},
		{name: "bad request timeout", mutate: func(rs *ReverseSearchSettings) {
			rs.RequestTimeout = "-1s"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			settings := Defaults()
			settings.ReverseSearch = base
			tc.mutate(&settings.ReverseSearch)
			if _, err := ResolveReverseSearch(settings, func(string) string { return "x" }); err == nil {
				t.Fatal("ResolveReverseSearch accepted invalid configuration")
			}
		})
	}
}

func TestDocumentRoundTripPreservesUnknownKeysAndSources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	input := `host = "auto"
unknown_top = "keep-me"

[reverse_search]
default_source = "custom"
cache = true

[[reverse_search.sources]]
name = "custom"
url = "https://example.test/search"

[reverse_search.sources.headers]
Authorization = "Bearer ${ENV:TOKEN}"
`
	if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := LoadDocument(path)
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	if err := document.Set("reverse_search.retries", 3); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := document.Set("host", "mirror"); err != nil {
		t.Fatalf("Set host: %v", err)
	}
	if err := document.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if reloaded.Host != "mirror" {
		t.Errorf("host = %q, want mirror", reloaded.Host)
	}
	if reloaded.ReverseSearch.Retries != 3 {
		t.Errorf("retries = %d, want 3", reloaded.ReverseSearch.Retries)
	}
	if reloaded.ReverseSearch.Sources[0].Headers["Authorization"] != "Bearer ${ENV:TOKEN}" {
		t.Errorf("header value was not preserved: %+v", reloaded.ReverseSearch.Sources)
	}

	// 未知键必须保留语义值。
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "keep-me") {
		t.Errorf("unknown key lost after save:\n%s", raw)
	}
	if !strings.Contains(string(raw), "reverse_search.sources") {
		t.Errorf("sources table lost after save:\n%s", raw)
	}
}

func TestDocumentSetCreatesNestedTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	document, err := LoadDocument(path)
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	if err := document.Set("reverse_search.retries", 5); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := document.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if loaded.ReverseSearch.Retries != 5 {
		t.Errorf("retries = %d, want 5", loaded.ReverseSearch.Retries)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("config file permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestDocumentDeleteRemovesKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("host = \"main\"\n[reverse_search]\nretries = 3\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	document, err := LoadDocument(path)
	if err != nil {
		t.Fatalf("LoadDocument: %v", err)
	}
	if err := document.Delete("reverse_search.retries"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := document.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if loaded.ReverseSearch.Retries != DefaultReverseSearchRetries {
		t.Errorf("retries = %d, want default %d after delete", loaded.ReverseSearch.Retries, DefaultReverseSearchRetries)
	}
}

func TestSaveFilePreservesUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("unknown_top = 42\nhost = \"auto\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	loaded.Host = "main"
	if err := SaveFile(path, loaded); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "unknown_top") {
		t.Errorf("SaveFile dropped the unknown key:\n%s", raw)
	}
	reloaded, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if reloaded.Host != "main" {
		t.Errorf("host = %q", reloaded.Host)
	}
}

func TestExpandEnvValueRejectsMultipleMissing(t *testing.T) {
	_, err := ExpandEnvValue("${ENV:A}${ENV:B}", func(string) string { return "" })
	if err == nil {
		t.Fatal("ExpandEnvValue accepted missing variables")
	}
}
