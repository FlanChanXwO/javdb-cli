package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// isolateHome 把 HOME 指向临时目录，避免配置命令测试污染真实本机状态。
func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(home))
	t.Setenv("HOMEPATH", strings.TrimPrefix(home, filepath.VolumeName(home)))
	return home
}

// executeConfig 用给定参数运行独立 config 命令树（无根 hook），返回输出缓冲。
func executeConfig(t *testing.T, args ...string) (bytes.Buffer, bytes.Buffer, error) {
	t.Helper()
	var out, errb bytes.Buffer
	streams := invocation.NewStreams(strings.NewReader(""), &out, &errb)
	streams.InIsTerminal = true
	command := New(streams)
	command.SetOut(&out)
	command.SetErr(&errb)
	command.SetArgs(args)
	return out, errb, command.Execute()
}

func TestConfigUnsetOnMissingConfigIsNoOp(t *testing.T) {
	home := isolateHome(t)
	out, errb, err := executeConfig(t, "unset", "host")
	if err != nil {
		t.Fatalf("unset on missing config error = %v", err)
	}
	if out.String() != "" || errb.String() != "" {
		t.Fatalf("unset on missing config produced output: stdout=%q stderr=%q", out.String(), errb.String())
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unset on missing config created file: %v", err)
	}
}

func TestConfigUnsetOnExistingFileResetsHostToAuto(t *testing.T) {
	isolateHome(t)
	if _, _, err := executeConfig(t, "set", "host", "main"); err != nil {
		t.Fatalf("set host error = %v", err)
	}
	if _, _, err := executeConfig(t, "unset", "host"); err != nil {
		t.Fatalf("unset host error = %v", err)
	}
	out, _, err := executeConfig(t, "get", "host")
	if err != nil {
		t.Fatalf("get host error = %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "auto" {
		t.Fatalf("host after unset = %q, want %q", got, "auto")
	}
}

func TestConfigSetCreatesPrivateFile(t *testing.T) {
	home := isolateHome(t)
	if _, _, err := executeConfig(t, "set", "lang", "zh"); err != nil {
		t.Fatalf("set lang error = %v", err)
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config.toml not created: %v", err)
	}
	out, _, err := executeConfig(t, "get", "lang")
	if err != nil {
		t.Fatalf("get lang error = %v", err)
	}
	if got := strings.TrimSpace(out.String()); got != "zh" {
		t.Fatalf("lang after set = %q, want %q", got, "zh")
	}
}

func TestConfigUnsetUnknownKeyErrorsOnMissingConfig(t *testing.T) {
	home := isolateHome(t)
	if _, _, err := executeConfig(t, "unset", "bogus"); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("unset bogus error = %v, want unknown key", err)
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("unset bogus created config: %v", statErr)
	}
}

func TestConfigGetUnknownKeyErrorsWithoutCreating(t *testing.T) {
	home := isolateHome(t)
	if _, _, err := executeConfig(t, "get", "bogus"); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("get bogus error = %v, want unknown key", err)
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("get bogus created config: %v", statErr)
	}
}

func TestConfigSetUnknownKeyErrorsWithoutCreating(t *testing.T) {
	home := isolateHome(t)
	if _, _, err := executeConfig(t, "set", "bogus", "x"); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("set bogus error = %v, want unknown key", err)
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("set bogus created config: %v", statErr)
	}
}

func TestConfigSetInvalidHostValueErrorsWithoutCreating(t *testing.T) {
	home := isolateHome(t)
	if _, _, err := executeConfig(t, "set", "host", "bogus"); err == nil || !strings.Contains(err.Error(), "host must be") {
		t.Fatalf("set host bogus error = %v, want host validation error", err)
	}
	path := filepath.Join(home, ".javdb-cli", "config.toml")
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("set host bogus created config: %v", statErr)
	}
}

func TestConfigReverseSearchScalarRoundTrip(t *testing.T) {
	isolateHome(t)
	out, _, err := executeConfig(t, "set", "reverse_search.retries", "5")
	if err != nil {
		t.Fatalf("set retries: %v", err)
	}
	_ = out
	out, _, err = executeConfig(t, "get", "reverse_search.retries")
	if err != nil {
		t.Fatalf("get retries: %v", err)
	}
	if strings.TrimSpace(out.String()) != "5" {
		t.Errorf("get retries = %q, want 5", out.String())
	}

	if _, _, err := executeConfig(t, "set", "reverse_search.cache", "false"); err != nil {
		t.Fatalf("set cache: %v", err)
	}
	out, _, err = executeConfig(t, "get", "reverse_search.cache")
	if err != nil {
		t.Fatalf("get cache: %v", err)
	}
	if strings.TrimSpace(out.String()) != "false" {
		t.Errorf("get cache = %q, want false", out.String())
	}

	if _, _, err := executeConfig(t, "set", "reverse_search.default_source", "custom"); err != nil {
		t.Fatalf("set default_source: %v", err)
	}
	out, _, err = executeConfig(t, "get", "reverse_search.default_source")
	if err != nil {
		t.Fatalf("get default_source: %v", err)
	}
	if strings.TrimSpace(out.String()) != "custom" {
		t.Errorf("get default_source = %q", out.String())
	}
}

func TestConfigReverseSearchUnsetFallsBackToDefaults(t *testing.T) {
	isolateHome(t)
	if _, _, err := executeConfig(t, "set", "reverse_search.default_source", "custom"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := executeConfig(t, "unset", "reverse_search.default_source"); err != nil {
		t.Fatal(err)
	}
	out, _, err := executeConfig(t, "get", "reverse_search.default_source")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "builtin" {
		t.Errorf("default_source after unset = %q, want builtin", out.String())
	}
}

func TestConfigReverseSearchKeyValidation(t *testing.T) {
	isolateHome(t)
	if _, _, err := executeConfig(t, "set", "reverse_search.retries", "abc"); err == nil {
		t.Fatal("set retries accepted a non-integer")
	}
	if _, _, err := executeConfig(t, "set", "reverse_search.cache", "maybe"); err == nil {
		t.Fatal("set cache accepted a non-boolean")
	}
	if _, _, err := executeConfig(t, "get", "reverse_search.unknown_key"); err == nil {
		t.Fatal("get accepted an unknown reverse_search key")
	}
	if _, _, err := executeConfig(t, "set", "reverse_search.sources", "[]"); err == nil {
		t.Fatal("set accepted the hand-edited sources array key")
	}
}

func TestConfigGetListsReverseSearchScalars(t *testing.T) {
	isolateHome(t)
	out, _, err := executeConfig(t, "get")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	for _, line := range []string{
		"reverse_search.default_source=builtin",
		"reverse_search.cache=true",
		"reverse_search.cache_ttl=720h",
		"reverse_search.retries=3",
		"reverse_search.retry_wait=30s",
		"reverse_search.request_timeout=60s",
	} {
		if !strings.Contains(out.String(), line) {
			t.Errorf("config get output lacks %q:\n%s", line, out.String())
		}
	}
}

// TestConfigGetStdinBatch 无 key + 非 TTY stdin：显式 JSONL 批处理输出 config_key 信封。
func TestConfigGetStdinBatch(t *testing.T) {
	isolateHome(t)
	streams := invocation.NewStreams(strings.NewReader("host\nlang\n"), &bytes.Buffer{}, &bytes.Buffer{})
	command := New(streams)
	command.SetArgs([]string{"get", "--jsonl"})
	if err := command.Execute(); err != nil {
		t.Fatalf("get batch: %v", err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines = %d:\n%s", len(lines), out)
	}
	if !strings.Contains(lines[0], `"kind":"config_key"`) || !strings.Contains(lines[0], `"ref":"host"`) {
		t.Errorf("first envelope = %s", lines[0])
	}
	if !strings.Contains(lines[0], `"value":"auto"`) {
		t.Errorf("first envelope lacks value: %s", lines[0])
	}
}

// TestConfigGetBatchRedactsProxyCredentials 管道信封不得泄漏 proxy 凭据。
func TestConfigGetBatchRedactsProxyCredentials(t *testing.T) {
	isolateHome(t)
	if _, _, err := executeConfig(t, "set", "https_proxy", "http://user:secret@proxy.example:8080"); err != nil {
		t.Fatal(err)
	}
	streams := invocation.NewStreams(strings.NewReader("https_proxy\n"), &bytes.Buffer{}, &bytes.Buffer{})
	command := New(streams)
	command.SetArgs([]string{"get", "--jsonl"})
	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}
	out := streams.Out.(*bytes.Buffer).String()
	if strings.Contains(out, "secret") {
		t.Fatalf("pipeline envelope leaks proxy credentials: %s", out)
	}
	if !strings.Contains(out, `"value":"***"`) {
		t.Errorf("proxy value should be redacted: %s", out)
	}
}

// TestConfigUnsetStdinBatch 非 TTY stdin key 批处理 unset。
func TestConfigUnsetStdinBatch(t *testing.T) {
	isolateHome(t)
	if _, _, err := executeConfig(t, "set", "lang", "zh-CN"); err != nil {
		t.Fatal(err)
	}
	streams := invocation.NewStreams(strings.NewReader("lang\n"), &bytes.Buffer{}, &bytes.Buffer{})
	command := New(streams)
	command.SetArgs([]string{"unset"})
	if err := command.Execute(); err != nil {
		t.Fatalf("unset batch: %v", err)
	}
	out, _, err := executeConfig(t, "get", "lang")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.String()) != "en" {
		t.Errorf("lang after batch unset = %q, want en", out.String())
	}
}
