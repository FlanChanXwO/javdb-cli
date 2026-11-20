package authstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
)

// isolateHome points the platform home lookups at a fresh temp directory so the
// auth store never leaks across tests or into a real user profile.
func isolateHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOMEDRIVE", filepath.VolumeName(dir))
	t.Setenv("HOMEPATH", strings.TrimPrefix(dir, filepath.VolumeName(dir)))
	return dir
}

func TestOpenCreatesDirectoryAndStore(t *testing.T) {
	home := isolateHome(t)
	fs, store, err := Open()
	if err != nil {
		t.Fatalf("Open error = %v", err)
	}
	if fs == nil || store == nil {
		t.Fatalf("Open returned nil fs/store")
	}
	dir := filepath.Join(home, ".javdb-cli")
	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Fatalf("config dir not created: %v", err)
	}
	if len(store.Accounts) != 0 {
		t.Fatalf("fresh store has accounts: %v", store.Accounts)
	}
}

func TestOpenPersistsAcrossCalls(t *testing.T) {
	isolateHome(t)
	fs, store, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	store.Upsert(auth.Account{UserID: 1, Username: "u", Token: "tok"}, true)
	if err := fs.Commit(store); err != nil {
		t.Fatal(err)
	}

	fs2, store2, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	if fs2 == nil {
		t.Fatal("reopen returned nil file store")
	}
	acc, err := store2.Default()
	if err != nil {
		t.Fatalf("default account: %v", err)
	}
	if acc.UserID != 1 || acc.Token != "tok" {
		t.Fatalf("reloaded account = %+v", acc)
	}
}
