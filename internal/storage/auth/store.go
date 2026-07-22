// Package auth stores multi-account JavDB credentials under ~/.javdb-cli/auth.json.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Account is one saved login (password stored by design).
type Account struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

// Store is the on-disk multi-account file.
type Store struct {
	DefaultUserID int64     `json:"default_user_id"`
	Accounts      []Account `json:"accounts"`
}

// ErrNotFound is returned when an account lookup fails.
var ErrNotFound = errors.New("account not found")

// Path is the default auth.json location (set by caller; empty → ~/.javdb-cli/auth.json).
// Kept as a package-level for tests via Open(path).

// Load reads path; missing file returns empty store.
func Load(path string) (*Store, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Store{}, nil
		}
		return nil, err
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save writes the store atomically with 0600 permissions where the platform
// supports POSIX permission bits.
func Save(path string, s *Store) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Default returns the default account, or ErrNotFound.
func (s *Store) Default() (Account, error) {
	if s.DefaultUserID == 0 {
		if len(s.Accounts) == 1 {
			return s.Accounts[0], nil
		}
		return Account{}, ErrNotFound
	}
	return s.Get(s.DefaultUserID)
}

// Get finds an account by user id.
func (s *Store) Get(userID int64) (Account, error) {
	for _, a := range s.Accounts {
		if a.UserID == userID {
			return a, nil
		}
	}
	return Account{}, fmt.Errorf("%w: %d", ErrNotFound, userID)
}

// Upsert inserts or replaces by user_id. If setDefault, updates default_user_id.
func (s *Store) Upsert(a Account, setDefault bool) {
	for i, existing := range s.Accounts {
		if existing.UserID == a.UserID {
			s.Accounts[i] = a
			if setDefault {
				s.DefaultUserID = a.UserID
			}
			return
		}
	}
	s.Accounts = append(s.Accounts, a)
	if setDefault || s.DefaultUserID == 0 {
		s.DefaultUserID = a.UserID
	}
}

// Remove deletes by user id. If it was default, clears or reassigns default.
func (s *Store) Remove(userID int64) error {
	idx := -1
	for i, a := range s.Accounts {
		if a.UserID == userID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("%w: %d", ErrNotFound, userID)
	}
	s.Accounts = append(s.Accounts[:idx], s.Accounts[idx+1:]...)
	if s.DefaultUserID == userID {
		s.DefaultUserID = 0
		if len(s.Accounts) > 0 {
			s.DefaultUserID = s.Accounts[0].UserID
		}
	}
	return nil
}

// Use sets the default user id.
func (s *Store) Use(userID int64) error {
	if _, err := s.Get(userID); err != nil {
		return err
	}
	s.DefaultUserID = userID
	return nil
}

// UpdateToken replaces the token for an account.
func (s *Store) UpdateToken(userID int64, token string) error {
	for i, a := range s.Accounts {
		if a.UserID == userID {
			s.Accounts[i].Token = token
			return nil
		}
	}
	return fmt.Errorf("%w: %d", ErrNotFound, userID)
}

// FileStore is a path-backed store with a mutex for CLI use.
type FileStore struct {
	Path string
	mu   sync.Mutex
}

// Open loads (or creates empty) the store at path.
func Open(path string) (*FileStore, *Store, error) {
	s, err := Load(path)
	if err != nil {
		return nil, nil, err
	}
	return &FileStore{Path: path}, s, nil
}

// Commit saves s to the file store path.
func (f *FileStore) Commit(s *Store) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return Save(f.Path, s)
}
