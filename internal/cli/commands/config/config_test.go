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
