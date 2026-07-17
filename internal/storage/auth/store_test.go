package auth

import (
	"os"
	"path/filepath"
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
	// mode
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", fi.Mode().Perm())
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
