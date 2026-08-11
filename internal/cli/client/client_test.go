package client

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
)

// isolateHome points the platform home lookups at a fresh temp directory so config
// and device state never leak across tests or into a real user profile.
func isolateHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(dir))
	t.Setenv("HOMEPATH", strings.TrimPrefix(dir, filepath.VolumeName(dir)))
	return dir
}

func TestNewBuildsClientWithoutNetwork(t *testing.T) {
	isolateHome(t)
	c, err := New(&invocation.RootOptions{}, "")
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	if c == nil {
		t.Fatal("New returned nil client")
	}
	if got := c.Token(); got != "" {
		t.Fatalf("Token() = %q, want empty", got)
	}
}

func TestNewRejectsInvalidHost(t *testing.T) {
	isolateHome(t)
	if _, err := New(&invocation.RootOptions{Host: "bogus"}, ""); err == nil {
		t.Fatal("New accepted an invalid host")
	}
}

func TestNewCreatesAndPersistsDeviceUUID(t *testing.T) {
	home := isolateHome(t)

	first, err := New(&invocation.RootOptions{}, "")
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	id1 := first.API().DeviceUUID()
	if id1 == "" {
		t.Fatal("DeviceUUID() is empty")
	}
	devicePath := filepath.Join(home, ".javdb-cli", "device_uuid")
	if data, err := os.ReadFile(devicePath); err != nil || strings.TrimSpace(string(data)) != id1 {
		t.Fatalf("device_uuid file = %q, %v; want %q", data, err, id1)
	}

	second, err := New(&invocation.RootOptions{}, "")
	if err != nil {
		t.Fatalf("second New error = %v", err)
	}
	if got := second.API().DeviceUUID(); got != id1 {
		t.Fatalf("reused DeviceUUID = %q, want %q", got, id1)
	}
}

func TestNewCarriesToken(t *testing.T) {
	isolateHome(t)
	c, err := New(&invocation.RootOptions{}, "tok")
	if err != nil {
		t.Fatalf("New error = %v", err)
	}
	if got := c.Token(); got != "tok" {
		t.Fatalf("Token() = %q, want tok", got)
	}
}
