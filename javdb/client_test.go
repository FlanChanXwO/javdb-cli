package javdb

import (
	"path/filepath"
	"testing"
)

func TestLoadOrCreateDeviceUUIDReusesPersistedValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "device_uuid")

	first, err := LoadOrCreateDeviceUUID(path)
	if err != nil {
		t.Fatalf("create device UUID: %v", err)
	}
	if first == "" {
		t.Fatal("created device UUID is empty")
	}

	second, err := LoadOrCreateDeviceUUID(path)
	if err != nil {
		t.Fatalf("reload device UUID: %v", err)
	}
	if second != first {
		t.Fatalf("reloaded device UUID = %q, want %q", second, first)
	}
}
