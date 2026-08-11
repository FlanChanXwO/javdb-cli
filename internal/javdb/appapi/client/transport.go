// Package client 持有签名 App API 的 HTTP transport 与公共请求处理。
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"

	"github.com/FlanChanXwO/javdb-cli/internal/javdb/appapi/model"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/protocol/httpx"
	"github.com/FlanChanXwO/javdb-cli/internal/javdb/protocol/signature"
)

const (
	AppVersion       = model.AppVersion
	AppVersionNumber = model.AppVersionNumber
	UserAgent        = model.UserAgent
	HostMirror       = model.HostMirror
	HostMain         = model.HostMain

	// defaultRetries 是普通业务请求的默认重试次数。
	defaultRetries = 2
)

// Auth actions that mean the bearer token is missing/invalid.
var authActions = map[string]bool{
	"JWTVerificationError": true,
	"Unauthorized":         true,
	"LoginRequired":        true,
	"TokenInvalid":         true,
	"TokenExpired":         true,
}

// Error 和 AuthRequired 通过 alias 保持与 model 及公开 SDK 的类型身份一致。
type Error = model.Error
type AuthRequired = model.AuthRequired

// Client talks to /api/v1..v4 with jdsignature.
type Client struct {
	http       *httpx.Client
	host       string
	token      string
	deviceUUID string
	lang       string
	public     map[string]string
	retries    int
}

// Options 是 model.Options 的兼容 alias。
type Options = model.Options

// New constructs a signed API client.
func New(opts Options) (*Client, error) {
	if opts.Host == "" {
		opts.Host = HostMirror
	}
	opts.Host = strings.TrimRight(opts.Host, "/")
	if opts.Lang == "" {
		opts.Lang = "en"
	}
	if opts.AppChannel == "" {
		opts.AppChannel = "official"
	}
	if opts.SystemVersion == "" {
		opts.SystemVersion = "13"
	}
	if opts.DeviceModel == "" {
		opts.DeviceModel = "Pixel 6"
	}
	if opts.DeviceName == "" {
		opts.DeviceName = "Pixel"
	}
	if opts.DeviceUUID == "" {
		opts.DeviceUUID = newDeviceUUID()
	}
	hc, err := httpx.New(httpx.Options{Proxy: opts.Proxy, Timeout: opts.Timeout})
	if err != nil {
		return nil, err
	}
	// nil 表示默认重试 2 次；显式 0 表示零重试（测速/探测场景）。
	retries := defaultRetries
	if opts.Retries != nil {
		retries = *opts.Retries
	}
	return &Client{
		http:       hc,
		host:       opts.Host,
		token:      opts.Token,
		deviceUUID: opts.DeviceUUID,
		lang:       opts.Lang,
		retries:    retries,
		public: map[string]string{
			"app_channel":        opts.AppChannel,
			"app_version":        AppVersion,
			"app_version_number": AppVersionNumber,
			"platform":           "android",
			"system_version":     opts.SystemVersion,
			"device_model":       opts.DeviceModel,
			"device_name":        opts.DeviceName,
			"device_uuid":        opts.DeviceUUID,
		},
	}, nil
}

// SetToken updates the bearer token.
func (c *Client) SetToken(token string) { c.token = token }

// Token returns the current bearer token.
func (c *Client) Token() string { return c.token }

// DeviceUUID returns the device id used in public params.
func (c *Client) DeviceUUID() string { return c.deviceUUID }

// Language returns the language used by request headers.
func (c *Client) Language() string { return c.lang }

// SetLanguage temporarily changes the language used by request headers.
func (c *Client) SetLanguage(lang string) { c.lang = lang }

// FetchMedia 获取未经过 App envelope 包装的媒体资源，供 media 包通过 callback 使用。
func (c *Client) FetchMedia(rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("invalid media URL")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, fmt.Errorf("unsupported media URL scheme %q", u.Scheme)
	}
	resp, err := c.http.Get(rawURL, map[string]string{"user-agent": UserAgent})
	if err != nil {
		return nil, fmt.Errorf("request media: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if closeErr := resp.Body.Close(); closeErr != nil {
			return nil, fmt.Errorf("close media response after HTTP %d: %w", resp.StatusCode, closeErr)
		}
		return nil, fmt.Errorf("media request returned HTTP %d", resp.StatusCode)
	}
	body, err := httpx.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("read media: %w", err)
	}
	return body, nil
}

func (c *Client) headers(ts int64) http.Header {
	h := http.Header{}
	h.Set("jdsignature", signature.Sign(ts))
	h.Set("accept-language", c.lang)
	h.Set("connection", "keep-alive")
	h.Set("user-agent", UserAgent)
	if c.token != "" {
		h.Set("authorization", "Bearer "+c.token)
	}
	// Order-ish: fhttp uses http.Header map; ok for this API.
	return h
}

func (c *Client) mergeParams(extra map[string]string) url.Values {
	q := url.Values{}
	for k, v := range c.public {
		q.Set(k, v)
	}
	for k, v := range extra {
		if v == "" {
			continue
		}
		q.Set(k, v)
	}
	return q
}

type envelope struct {
	Success any             `json:"success"`
	Action  string          `json:"action"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func successTruthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	case json.Number:
		n, _ := t.Int64()
		return n != 0
	default:
		return v != nil
	}
}

// doContext 发送一次签名请求。retries 为 0 时单次尝试即返回，不会因退避睡眠污染测速样本；
// 退避期间观察 ctx 取消，避免重试在取消后继续空等。
func (c *Client) doContext(ctx context.Context, method, path string, extra map[string]string, retries int) (json.RawMessage, error) {
	var last error
	for attempt := 0; attempt <= retries; attempt++ {
		ts := time.Now().Unix()
		params := c.mergeParams(extra)
		u := c.host + path
		var resp *http.Response
		var err error
		hdrs := c.headers(ts)
		// Convert header to map for httpx helpers when needed
		hm := map[string]string{}
		for k, vs := range hdrs {
			if len(vs) > 0 {
				hm[k] = vs[0]
			}
		}

		switch method {
		case http.MethodGet, http.MethodDelete:
			full := u
			if enc := params.Encode(); enc != "" {
				full = u + "?" + enc
			}
			if method == http.MethodGet {
				resp, err = c.http.GetWithContext(ctx, full, hm)
			} else {
				resp, err = c.http.DeleteWithContext(ctx, full, hm)
			}
		default: // POST form
			resp, err = c.http.PostFormWithContext(ctx, u, params, hm)
		}
		if err != nil {
			last = err
			if attempt < retries {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(attempt+1) * 500 * time.Millisecond):
				}
				continue
			}
			return nil, err
		}
		body, err := httpx.ReadAll(resp)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("http %d: %s", resp.StatusCode, truncate(string(body), 200))
		}
		var env envelope
		if err := json.Unmarshal(body, &env); err != nil {
			return nil, fmt.Errorf("decode: %w; body=%s", err, truncate(string(body), 200))
		}
		if !successTruthy(env.Success) {
			ae := Error{Action: env.Action, Message: env.Message}
			if authActions[env.Action] {
				return nil, &AuthRequired{API: ae}
			}
			return nil, &ae
		}
		if len(env.Data) == 0 || string(env.Data) == "null" {
			return json.RawMessage("{}"), nil
		}
		return env.Data, nil
	}
	return nil, last
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// GetJSON performs a signed GET and unmarshals data into dest (optional).
func (c *Client) GetJSON(path string, params map[string]string, dest any) error {
	return c.GetJSONContext(context.Background(), path, params, dest)
}

// GetJSONContext performs a signed GET with an explicit context.
func (c *Client) GetJSONContext(ctx context.Context, path string, params map[string]string, dest any) error {
	raw, err := c.doContext(ctx, http.MethodGet, path, params, c.retries)
	if err != nil {
		return err
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(raw, dest)
}

// PostFormJSON performs a signed POST form and unmarshals data.
func (c *Client) PostFormJSON(path string, form map[string]string, dest any) error {
	return c.PostFormJSONContext(context.Background(), path, form, dest)
}

// PostFormJSONContext performs a signed POST form with an explicit context.
func (c *Client) PostFormJSONContext(ctx context.Context, path string, form map[string]string, dest any) error {
	raw, err := c.doContext(ctx, http.MethodPost, path, form, c.retries)
	if err != nil {
		return err
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(raw, dest)
}

// DeleteJSON performs a signed DELETE.
func (c *Client) DeleteJSON(path string, params map[string]string, dest any) error {
	return c.DeleteJSONContext(context.Background(), path, params, dest)
}

// DeleteJSONContext performs a signed DELETE with an explicit context.
func (c *Client) DeleteJSONContext(ctx context.Context, path string, params map[string]string, dest any) error {
	raw, err := c.doContext(ctx, http.MethodDelete, path, params, c.retries)
	if err != nil {
		return err
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(raw, dest)
}
