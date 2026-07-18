package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/FlanChanXwO/javdb-cli/internal/appapi"
	"github.com/FlanChanXwO/javdb-cli/javdb"
)

// withAuthedClient runs fn with a client carrying the default account token.
// On AuthRequired: if runtime.AutoRelogin, re-login with saved password once and retry;
// otherwise return a clear error.
func withAuthedClient(rf *rootFlags, aio *appIO, fn func(*javdb.Client) error) error {
	rt, err := loadRuntime(rf)
	if err != nil {
		return err
	}
	fs, store, err := openAuth()
	if err != nil {
		return err
	}
	acc, err := store.Default()
	if err != nil {
		return fmt.Errorf("no default account; run: javdb auth login")
	}
	if acc.Token == "" {
		return fmt.Errorf("default account has no token; run: javdb auth login")
	}
	c, err := newClient(rt, acc.Token)
	if err != nil {
		return err
	}
	err = fn(c)
	if err == nil {
		return nil
	}
	var ar *appapi.AuthRequired
	if !errors.As(err, &ar) {
		// also check javdb alias
		var ar2 *javdb.AuthRequired
		if !errors.As(err, &ar2) {
			return err
		}
	}
	if !rt.AutoRelogin {
		return fmt.Errorf("token expired or invalid; run: javdb auth login (or: javdb config set auto_relogin true)")
	}
	if acc.Password == "" {
		return fmt.Errorf("token expired and no saved password; run: javdb auth login")
	}
	if aio != nil && aio.err != nil {
		fmt.Fprintln(aio.err, "缓存 token 已失效，重新登录…")
	}
	// re-login
	c2, err := newClient(rt, "")
	if err != nil {
		return err
	}
	tok, err := c2.Login(context.Background(), acc.Username, acc.Password)
	if err != nil {
		return fmt.Errorf("auto re-login failed: %w", err)
	}
	// preserve user id; update token in store
	if err := store.UpdateToken(acc.UserID, tok); err != nil {
		return err
	}
	if err := fs.Commit(store); err != nil {
		return err
	}
	c2.SetToken(tok)
	return fn(c2)
}
