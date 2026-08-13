package client

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// WithRequiredAuth 使用默认账户 token 执行 fn；按配置决定是否自动重新登录。
// 本地账号/token 前置检查先于线路选择：断网或代理不可用时仍先报"请先登录"类错误，
// 不会因自动选线失败而掩盖认证缺失。错误文本与既有 WithAuthedClient 逐字一致；
// errOut == nil 时不写提示。
func WithRequiredAuth(options *invocation.RootOptions, errOut io.Writer, fn func(*javdb.Client) error) error {
	rt, err := resolveRuntime(options)
	if err != nil {
		return err
	}
	fs, store, err := authstore.Open()
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
	// host/proxy 无副作用校验与本地检查都通过后再创建基线配置，随后执行可能失败的网络选线。
	baseURL, err := ensureConfigAndBaseURL(rt, productionAutoHost)
	if err != nil {
		return err
	}
	c, err := buildClient(rt, baseURL, acc.Token)
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
	if errOut != nil {
		fmt.Fprintln(errOut, "缓存 token 已失效，重新登录…")
	}
	// re-login
	c2, err := buildClient(rt, baseURL, "")
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

// WithOptionalAuth 在有 token 时携带默认账户，否则匿名执行；认证失败时保留原有匿名重试。
// 本地 token 读取先于线路选择，避免只读/匿名命令在离线时被选线失败阻断。
func WithOptionalAuth(options *invocation.RootOptions, errOut io.Writer, fn func(*javdb.Client) error) error {
	rt, err := resolveRuntime(options)
	if err != nil {
		return err
	}
	token := ""
	if _, store, err := authstore.Open(); err == nil {
		if acc, aerr := store.Default(); aerr == nil {
			token = acc.Token
		}
	}
	baseURL, err := ensureConfigAndBaseURL(rt, productionAutoHost)
	if err != nil {
		return err
	}
	c, err := buildClient(rt, baseURL, token)
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
	if errOut != nil {
		fmt.Fprintln(errOut, "token 无效，改用匿名请求…")
	}
	c2, err := buildClient(rt, baseURL, "")
	if err != nil {
		return err
	}
	return fn(c2)
}

// NewWithDefaultToken 解析运行时与默认账号 token 并构造公开 SDK client；
// 供管道批处理调用方每次批量只构造一次。解析失败显式返回错误。
func NewWithDefaultToken(options *invocation.RootOptions) (*javdb.Client, error) {
	rt, err := resolveRuntime(options)
	if err != nil {
		return nil, err
	}
	token := ""
	if _, store, err := authstore.Open(); err == nil {
		if acc, aerr := store.Default(); aerr == nil {
			token = acc.Token
		}
	}
	baseURL, err := ensureConfigAndBaseURL(rt, productionAutoHost)
	if err != nil {
		return nil, err
	}
	return buildClient(rt, baseURL, token)
}
