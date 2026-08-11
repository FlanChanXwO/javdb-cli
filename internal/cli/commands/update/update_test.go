package update

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

func newCommand(t *testing.T) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var out, errb bytes.Buffer
	streams := invocation.NewStreams(strings.NewReader(""), &out, &errb)
	cmd := New(&invocation.RootOptions{}, streams)
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	return cmd, &out, &errb
}

func TestUpdateRejectsJSONInstallOutput(t *testing.T) {
	cmd, _, _ := newCommand(t)
	cmd.SetArgs([]string{"--json"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--json is only supported with --check") {
		t.Fatalf("error = %v, want --json requires --check", err)
	}
}

func TestUpdateHelpShowsExpectedText(t *testing.T) {
	cmd, out, _ := newCommand(t)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("help error = %v", err)
	}
	if !strings.Contains(out.String(), "Check for or install updates") {
		t.Fatalf("update help = %q", out.String())
	}
}

func TestResolveProxyUsesConfigPriority(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(home))
	t.Setenv("HOMEPATH", strings.TrimPrefix(home, filepath.VolumeName(home)))

	// 无配置文件 → 默认 proxy 为空；flag 传入时直接采用。
	if got, err := resolveProxy(&invocation.RootOptions{}); err != nil || got != "" {
		t.Fatalf("resolveProxy(empty) = %q, %v", got, err)
	}
	if got, err := resolveProxy(&invocation.RootOptions{Proxy: "http://proxy.invalid"}); err != nil || got != "http://proxy.invalid" {
		t.Fatalf("resolveProxy(flag) = %q, %v", got, err)
	}
}

func TestNewProductionCoordinatorOffline(t *testing.T) {
	coordinator, err := newProductionCoordinator("", io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("newProductionCoordinator error = %v", err)
	}
	if coordinator == nil {
		t.Fatal("newProductionCoordinator returned nil")
	}
}
