package auth

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUpsertUseRemove(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")
	s := &Store{}
	s.Upsert(Account{UserID: 1, Username: "a", Password: "p", Token: "t1"}, true)
	s.Upsert(Account{UserID: 2, Username: "b", Password: "p", Token: "t2"}, false)
	if s.DefaultUserID != 1 {
		t.Fatalf("default=%d", s.DefaultUserID)
	}
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	// Windows does not expose POSIX permission bits; os.WriteFile cannot enforce
	// 0600 there, while Unix targets must retain the credential-file contract.
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Fatalf("mode=%o", fi.Mode().Perm())
		}
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Accounts) != 2 {
		t.Fatalf("n=%d", len(loaded.Accounts))
	}
	if err := loaded.Use(2); err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultUserID != 2 {
		t.Fatal("use failed")
	}
	if err := loaded.Remove(2); err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultUserID != 1 {
		t.Fatalf("after remove default=%d", loaded.DefaultUserID)
	}
}

func TestResolveAndLoadErrors(t *testing.T) {
	single := &Store{Accounts: []Account{{UserID: 7, Username: "single"}}}
	account, err := single.Default()
	if err != nil || account.UserID != 7 {
		t.Fatalf("single default = %+v, %v", account, err)
	}
	if _, err := single.Get(9); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing account error = %v", err)
	}

	dir := t.TempDir()
	missing, err := Load(filepath.Join(dir, "missing.json"))
	if err != nil || missing == nil || len(missing.Accounts) != 0 {
		t.Fatalf("missing store = %+v, %v", missing, err)
	}
	invalidPath := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(invalidPath); err == nil {
		t.Fatal("invalid auth JSON unexpectedly loaded")
	}
}
