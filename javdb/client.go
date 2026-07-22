// Package javdb is the public Go SDK for the JavDB app JSON API.
package javdb

import (
	"context"
	"time"

	"github.com/FlanChanXwO/javdb-cli/internal/config"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi"
)

// Host constants.
const (
	HostMirror = config.HostMirror
	HostMain   = config.HostMain
)

// Client is a concrete app-API client.
type Client struct {
	api *appapi.Client
}

// Option configures New.
type Option func(*options)

type options struct {
	host       string
	token      string
	proxy      string
	deviceUUID string
	timeout    time.Duration
	lang       string
}

// WithHost sets logical host (mirror|main) or absolute base URL.
func WithHost(host string) Option {
	return func(o *options) { o.host = host }
}

// WithToken sets the bearer token.
func WithToken(token string) Option {
	return func(o *options) { o.token = token }
}

// WithProxy sets an HTTP(S) proxy URL.
func WithProxy(proxy string) Option {
	return func(o *options) { o.proxy = proxy }
}

// WithDeviceUUID sets the app device_uuid public param.
func WithDeviceUUID(id string) Option {
	return func(o *options) { o.deviceUUID = id }
}

// WithTimeout sets request timeout.
func WithTimeout(d time.Duration) Option {
	return func(o *options) { o.timeout = d }
}

// WithLang sets Accept-Language / app lang.
func WithLang(lang string) Option {
	return func(o *options) { o.lang = lang }
}

// New builds a Client.
func New(opts ...Option) (*Client, error) {
	o := options{
		host:    HostMirror,
		timeout: 20 * time.Second,
		lang:    "en",
	}
	for _, fn := range opts {
		fn(&o)
	}
	base := o.host
	if u, ok := config.HostURLs[o.host]; ok {
		base = u
	}
	api, err := appapi.New(appapi.Options{
		Host:       base,
		Token:      o.token,
		DeviceUUID: o.deviceUUID,
		Proxy:      o.proxy,
		Timeout:    o.timeout,
		Lang:       o.lang,
	})
	if err != nil {
		return nil, err
	}
	return &Client{api: api}, nil
}

// Token returns the current bearer token.
func (c *Client) Token() string { return c.api.Token() }

// SetToken updates the bearer token.
func (c *Client) SetToken(token string) { c.api.SetToken(token) }

// LoadOrCreateDeviceUUID loads the stable device identifier at path or creates
// one when the file is absent. CLI callers use it before constructing a client;
// SDK callers may use it when they need the same identifier across processes.
func LoadOrCreateDeviceUUID(path string) (string, error) {
	return appapi.LoadOrCreateDeviceUUID(path)
}

// Login authenticates and stores the token on the client.
// ctx is reserved for future cancellation; the underlying client is not yet context-aware.
func (c *Client) Login(ctx context.Context, username, password string) (string, error) {
	_ = ctx
	return c.api.Login(username, password)
}

// ResolveUserID returns the numeric user id (and optional display name) for the current token.
func (c *Client) ResolveUserID(ctx context.Context) (userID int64, username string, err error) {
	_ = ctx
	return c.api.ResolveUserID("")
}

// API 为兼容旧调用方返回底层 adapter。该返回值不是稳定的外部 SDK 契约；新代码应优先使用 typed 方法。
// Deprecated: use the documented Client methods instead.
func (c *Client) API() *appapi.Client { return c.api }
