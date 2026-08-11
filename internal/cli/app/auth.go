package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// withAuthedClient runs fn with a client carrying the default account token.
// On AuthRequired: if runtime.AutoRelogin, re-login with saved password once and retry;
// otherwise return a clear error.
// WithAuthedClient 使用默认账户 token 执行 fn；按配置决定是否自动重新登录。
func WithAuthedClient(flags *Flags, aio *IO, fn func(*javdb.Client) error) error {
	rt, err := LoadRuntime(flags)
	if err != nil {
		return err
	}
	fs, store, err := OpenAuth()
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
	c, err := NewClient(rt, acc.Token)
	if err != nil {
		return err
	}
	err = fn(c)
	if err == nil {
		return nil
	}
	var authRequired *javdb.AuthRequired
	if !errors.As(err, &authRequired) {
		return err
	}
	if !rt.AutoRelogin {
		return fmt.Errorf("token expired or invalid; run: javdb auth login (or: javdb config set auto_relogin true)")
	}
	if acc.Password == "" {
		return fmt.Errorf("token expired and no saved password; run: javdb auth login")
	}
	if aio != nil && aio.Err != nil {
		fmt.Fprintln(aio.Err, "缓存 token 已失效，重新登录…")
	}
	// re-login
	c2, err := NewClient(rt, "")
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

// withOptionalAuthClient runs fn with the default account token when one is
// saved, otherwise anonymously. It never fails upfront for missing auth.
// If the server rejects the token (AuthRequired), it retries fn once with an
// anonymous client so read-only discovery still works.
// WithOptionalAuthClient 在有 token 时携带默认账户，否则匿名执行；认证失败时保留原有匿名重试。
func WithOptionalAuthClient(flags *Flags, aio *IO, fn func(*javdb.Client) error) error {
	rt, err := LoadRuntime(flags)
	if err != nil {
		return err
	}
	token := ""
	if _, store, err := OpenAuth(); err == nil {
		if acc, aerr := store.Default(); aerr == nil {
			token = acc.Token
		}
	}
	c, err := NewClient(rt, token)
	if err != nil {
		return err
	}
	err = fn(c)
	if err == nil {
		return nil
	}
	var authRequired *javdb.AuthRequired
	if token == "" || !errors.As(err, &authRequired) {
		return err
	}
	if aio != nil && aio.Err != nil {
		fmt.Fprintln(aio.Err, "token 无效，改用匿名请求…")
	}
	c2, err := NewClient(rt, "")
	if err != nil {
		return err
	}
	return fn(c2)
}
