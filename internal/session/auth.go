package session

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"

	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/logging"
	"github.com/FlanChanXwO/javdb-cli/internal/session/authstore"
	"github.com/FlanChanXwO/javdb-cli/internal/storage/auth"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// Options owns flags and the authentication state of one CLI invocation.
type Options struct {
	Logger *slog.Logger
	Proxy  string
	Host   string
	mu     sync.Mutex
	state  *authState
}

type authState struct {
	logger    *slog.Logger
	mu        sync.Mutex
	account   auth.Account
	store     *auth.Store
	file      *auth.FileStore
	auto      bool
	newClient func() (*Client, error)
	started   bool
	recovered bool
	done      chan struct{}
	token     string
	err       error
}

// Client adds the CLI authentication lifecycle to the public SDK client.
// Network operations still use the public SDK; each rejected operation is replayed once.
type Client struct {
	*javdb.Client
	state *authState
}

// LoginError keeps the user action separate from the complete upstream cause.
type LoginError struct {
	Key   string
	Cause error
}

func (e *LoginError) MessageKey() string { return e.Key }
func (e *LoginError) Error() string {
	if e.Cause != nil {
		return i18n.T(e.Key) + " " + e.Cause.Error()
	}
	return i18n.T(e.Key)
}
func (e *LoginError) Unwrap() error { return e.Cause }

func (o *Options) accountState() (*authState, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.state != nil {
		return o.state, nil
	}
	file, store, err := authstore.Open()
	if err != nil {
		return nil, err
	}
	account, err := store.Default()
	if err != nil && !errors.Is(err, auth.ErrNotFound) {
		return nil, err
	}
	rt, err := resolveRuntime(o)
	if err != nil {
		return nil, err
	}
	logger := o.Logger
	if logger == nil {
		logger, _ = logging.New(io.Discard, "info", "text")
	}
	o.state = &authState{logger: logger, account: account, file: file, store: store, auto: rt.AutoRelogin, token: account.Token, newClient: func() (*Client, error) { return New(o, "") }}
	return o.state, nil
}

// NewWithDefaultToken builds a client using the default account of this invocation.
func NewWithDefaultToken(options *Options) (*Client, error) {
	state, err := options.accountState()
	if err != nil {
		return nil, err
	}
	state.mu.Lock()
	token := state.token
	state.mu.Unlock()
	c, err := New(options, token)
	if err != nil {
		return nil, err
	}
	c.state = state
	return c, nil
}

func (c *Client) requireAccount() error {
	if c.Token() == "" {
		return &LoginError{Key: "auth.required"}
	}
	return nil
}

func (c *Client) execute(ctx context.Context, fn func() error) error {
	hadRecovery := false
	if c.state != nil {
		c.state.mu.Lock()
		hadRecovery = c.state.recovered
		current := c.state.token
		c.state.mu.Unlock()
		if hadRecovery {
			c.SetToken(current)
		}
	}
	token := c.Token()
	err := fn()
	if err == nil {
		return nil
	}
	var presented *LoginError
	if errors.As(err, &presented) {
		return err
	}
	var rejected *javdb.AuthRequired
	if !errors.As(err, &rejected) {
		return err
	}
	if token == "" {
		return &LoginError{Key: "auth.required", Cause: err}
	}
	if c.state == nil || !c.state.auto || hadRecovery {
		return &LoginError{Key: "auth.invalid", Cause: err}
	}
	if c.state.account.Password == "" {
		return &LoginError{Key: "auth.no_password", Cause: err}
	}
	if refreshErr := c.refresh(ctx); refreshErr != nil {
		return &LoginError{Key: "auth.retry_failed", Cause: errors.Join(err, refreshErr)}
	}
	c.state.mu.Lock()
	fresh := c.state.token
	c.state.mu.Unlock()
	c.SetToken(fresh)
	if retryErr := fn(); retryErr != nil {
		if errors.As(retryErr, &rejected) {
			return &LoginError{Key: "auth.invalid", Cause: retryErr}
		}
		return retryErr
	}
	return nil
}

func (c *Client) refresh(ctx context.Context) error {
	s := c.state
	s.mu.Lock()
	if s.started {
		done := s.done
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
		}
		s.mu.Lock()
		err := s.err
		s.mu.Unlock()
		return err
	}
	s.started = true
	s.done = make(chan struct{})
	s.mu.Unlock()
	// Login uses a separate client, so no shared request observes a transient empty token.
	logErr := logging.Emit(s.logger, slog.LevelInfo, i18n.T("auth.retry"))
	login, err := s.newClient()
	if logErr != nil {
		err = logErr
	}
	token := ""
	if err == nil {
		token, err = login.Login(ctx, s.account.Username, s.account.Password)
	}
	if err == nil {
		err = s.store.UpdateToken(s.account.UserID, token)
		if err == nil {
			err = s.file.Commit(s.store)
		}
	}
	s.mu.Lock()
	s.err = err
	if err == nil {
		s.token = token
		s.recovered = true
	}
	close(s.done)
	s.mu.Unlock()
	return err
}

// WithRequiredAuth performs the local account check before the operation.
func WithRequiredAuth(options *Options, errOut io.Writer, fn func(*Client) error) error {
	if options.Logger == nil && errOut != nil {
		options.Logger, _ = logging.New(errOut, "info", "text")
	}
	c, err := NewRequired(options)
	if err != nil {
		return err
	}
	return c.execute(context.Background(), func() error { return fn(c) })
}

// WithOptionalAuth permits anonymous requests and honors explicit auto-relogin.
func WithOptionalAuth(options *Options, errOut io.Writer, fn func(*Client) error) error {
	if options.Logger == nil && errOut != nil {
		options.Logger, _ = logging.New(errOut, "info", "text")
	}
	c, err := NewWithDefaultToken(options)
	if err != nil {
		return err
	}
	return c.execute(context.Background(), func() error { return fn(c) })
}

// NewRequired checks local credentials before network route selection.
func NewRequired(options *Options) (*Client, error) {
	state, err := options.accountState()
	if err != nil {
		return nil, err
	}
	if state.account.Token == "" {
		return nil, &LoginError{Key: "auth.required"}
	}
	return NewWithDefaultToken(options)
}
