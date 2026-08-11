package auth

import "fmt"

// Upsert 按 user_id 插入或替换账户；setDefault 为 true 时同步默认账户。
func (s *Store) Upsert(account Account, setDefault bool) {
	for i, existing := range s.Accounts {
		if existing.UserID == account.UserID {
			s.Accounts[i] = account
			if setDefault {
				s.DefaultUserID = account.UserID
			}
			return
		}
	}
	s.Accounts = append(s.Accounts, account)
	if setDefault || s.DefaultUserID == 0 {
		s.DefaultUserID = account.UserID
	}
}

// Remove 删除账户；删除默认账户时沿用原有的首账户重选规则。
func (s *Store) Remove(userID int64) error {
	index := -1
	for i, account := range s.Accounts {
		if account.UserID == userID {
			index = i
			break
		}
	}
	if index < 0 {
		return fmt.Errorf("%w: %d", ErrNotFound, userID)
	}
	s.Accounts = append(s.Accounts[:index], s.Accounts[index+1:]...)
	if s.DefaultUserID == userID {
		s.DefaultUserID = 0
		if len(s.Accounts) > 0 {
			s.DefaultUserID = s.Accounts[0].UserID
		}
	}
	return nil
}

// Use 将指定账户设为默认账户。
func (s *Store) Use(userID int64) error {
	if _, err := s.Get(userID); err != nil {
		return err
	}
	s.DefaultUserID = userID
	return nil
}

// UpdateToken 替换指定账户的 token。
func (s *Store) UpdateToken(userID int64, token string) error {
	for i, account := range s.Accounts {
		if account.UserID == userID {
			s.Accounts[i].Token = token
			return nil
		}
	}
	return fmt.Errorf("%w: %d", ErrNotFound, userID)
}
