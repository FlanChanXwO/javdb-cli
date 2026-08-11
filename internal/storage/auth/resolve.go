package auth

import "fmt"

// Default 返回默认账户；未设置默认账户时仅在恰有一个账户的情况下推断。
func (s *Store) Default() (Account, error) {
	if s.DefaultUserID == 0 {
		if len(s.Accounts) == 1 {
			return s.Accounts[0], nil
		}
		return Account{}, ErrNotFound
	}
	return s.Get(s.DefaultUserID)
}

// Get 按 user id 查找账户。
func (s *Store) Get(userID int64) (Account, error) {
	for _, account := range s.Accounts {
		if account.UserID == userID {
			return account, nil
		}
	}
	return Account{}, fmt.Errorf("%w: %d", ErrNotFound, userID)
}
