// Package appapi is the signed JavDB app JSON API client.
package appapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"

	"github.com/FlanChanXwO/javdb-cli/internal/httpx"
	"github.com/FlanChanXwO/javdb-cli/internal/signature"
)

const (
	AppVersion       = "1.9.28"
	AppVersionNumber = "10928"
	UserAgent        = "Dart/3.4 (dart:io)"

	HostMirror = "https://jdforrepam.com"
	HostMain   = "https://javdb.com"
)

// Auth actions that mean the bearer token is missing/invalid.
var authActions = map[string]bool{
	"JWTVerificationError": true,
	"Unauthorized":         true,
	"LoginRequired":        true,
	"TokenInvalid":         true,
	"TokenExpired":         true,
}

// Error is a server-side API failure (success:0).
type Error struct {
	Action  string
	Message string
}

func (e *Error) Error() string {
	if e.Action == "" && e.Message == "" {
		return "api error"
	}
	return fmt.Sprintf("%s: %s", e.Action, e.Message)
}

// AuthRequired means the caller should re-login.
type AuthRequired struct {
	API Error
}

func (e *AuthRequired) Error() string { return e.API.Error() }

func (e *AuthRequired) Unwrap() error { return &e.API }

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

// Options for New.
type Options struct {
	Host       string // full base URL
	Token      string
	DeviceUUID string
	Proxy      string
	Timeout    time.Duration
	Lang       string
	// Device profile (optional overrides)
	AppChannel    string
	SystemVersion string
	DeviceModel   string
	DeviceName    string
}

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
	return &Client{
		http:       hc,
		host:       opts.Host,
		token:      opts.Token,
		deviceUUID: opts.DeviceUUID,
		lang:       opts.Lang,
		retries:    2,
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

func (c *Client) do(method, path string, extra map[string]string) (json.RawMessage, error) {
	var last error
	for attempt := 0; attempt <= c.retries; attempt++ {
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
				resp, err = c.http.Get(full, hm)
			} else {
				resp, err = c.http.Delete(full, hm)
			}
		default: // POST form
			resp, err = c.http.PostForm(u, params, hm)
		}
		if err != nil {
			last = err
			if attempt < c.retries {
				time.Sleep(time.Duration(attempt+1) * 500 * time.Millisecond)
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
	raw, err := c.do(http.MethodGet, path, params)
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
	raw, err := c.do(http.MethodPost, path, form)
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
	raw, err := c.do(http.MethodDelete, path, params)
	if err != nil {
		return err
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(raw, dest)
}

// --- auth / bootstrap -------------------------------------------------------

// LoginResponse is the data payload of POST /api/v1/sessions.
type LoginResponse struct {
	Token       string          `json:"token"`
	AccessToken string          `json:"access_token"`
	User        json.RawMessage `json:"user"`
	// catch-all for probing
	Raw map[string]json.RawMessage `json:"-"`
}

// Login posts username/password and stores the bearer token on the client.
// Returns the token string. Does not resolve user id by itself.
func (c *Client) Login(username, password string) (string, error) {
	var data map[string]json.RawMessage
	if err := c.PostFormJSON("/api/v1/sessions", map[string]string{
		"username": username,
		"password": password,
	}, &data); err != nil {
		return "", err
	}
	token := rawString(data, "token")
	if token == "" {
		token = rawString(data, "access_token")
	}
	if token == "" {
		return "", &Error{Action: "NoToken", Message: "login response had no token"}
	}
	c.token = token
	return token, nil
}

// Startup returns GET /api/v1/startup data.
func (c *Client) Startup() (map[string]json.RawMessage, error) {
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/startup", nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// Users returns GET /api/v1/users data (profile).
func (c *Client) Users() (map[string]json.RawMessage, error) {
	var data map[string]json.RawMessage
	if err := c.GetJSON("/api/v1/users", nil, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// ResolveUserID tries JWT claims, then /users, then /startup.user.
func (c *Client) ResolveUserID(token string) (int64, string, error) {
	if token == "" {
		token = c.token
	}
	if id, name, ok := userIDFromJWT(token); ok {
		return id, name, nil
	}
	// /users
	if data, err := c.Users(); err == nil {
		if id, name, ok := userIDFromMap(data); ok {
			return id, name, nil
		}
		// nested user
		if u, ok := data["user"]; ok {
			var nested map[string]json.RawMessage
			if json.Unmarshal(u, &nested) == nil {
				if id, name, ok := userIDFromMap(nested); ok {
					return id, name, nil
				}
			}
		}
	}
	// startup
	if data, err := c.Startup(); err == nil {
		if u, ok := data["user"]; ok {
			var nested map[string]json.RawMessage
			if json.Unmarshal(u, &nested) == nil {
				if id, name, ok := userIDFromMap(nested); ok {
					return id, name, nil
				}
			}
		}
	}
	return 0, "", &Error{Action: "NoUserID", Message: "could not resolve numeric user id from token/users/startup"}
}

func rawString(m map[string]json.RawMessage, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if json.Unmarshal(v, &s) == nil {
		return s
	}
	return strings.Trim(string(v), `"`)
}

func userIDFromMap(m map[string]json.RawMessage) (int64, string, bool) {
	// id fields
	for _, k := range []string{"id", "user_id", "uid"} {
		if v, ok := m[k]; ok {
			if id, ok := parseID(v); ok {
				name := rawString(m, "username")
				if name == "" {
					name = rawString(m, "email")
				}
				if name == "" {
					name = rawString(m, "name")
				}
				return id, name, true
			}
		}
	}
	return 0, "", false
}

func parseID(raw json.RawMessage) (int64, bool) {
	var n int64
	if json.Unmarshal(raw, &n) == nil && n != 0 {
		return n, true
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil && n != 0 {
			return n, true
		}
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil && f != 0 {
		return int64(f), true
	}
	return 0, false
}

// userIDFromJWT decodes the middle segment without verifying the signature.
func userIDFromJWT(token string) (int64, string, bool) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return 0, "", false
	}
	payload, err := decodeSegment(parts[1])
	if err != nil {
		return 0, "", false
	}
	var claims map[string]any
	if json.Unmarshal(payload, &claims) != nil {
		return 0, "", false
	}
	// common claim keys
	for _, k := range []string{"user_id", "uid", "id", "sub"} {
		if v, ok := claims[k]; ok {
			switch t := v.(type) {
			case float64:
				if t != 0 {
					return int64(t), claimString(claims, "username", "email", "name"), true
				}
			case string:
				if n, err := strconv.ParseInt(t, 10, 64); err == nil && n != 0 {
					return n, claimString(claims, "username", "email", "name"), true
				}
			}
		}
	}
	// nested user
	if u, ok := claims["user"].(map[string]any); ok {
		for _, k := range []string{"id", "user_id", "uid"} {
			if v, ok := u[k]; ok {
				switch t := v.(type) {
				case float64:
					if t != 0 {
						return int64(t), claimString(u, "username", "email", "name"), true
					}
				case string:
					if n, err := strconv.ParseInt(t, 10, 64); err == nil && n != 0 {
						return n, claimString(u, "username", "email", "name"), true
					}
				}
			}
		}
	}
	return 0, "", false
}

func claimString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return ""
}

func decodeSegment(seg string) ([]byte, error) {
	// base64url without padding
	s := seg
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	return decodeStd(s)
}

// avoid importing encoding/base64 at top for clarity — use std
func decodeStd(s string) ([]byte, error) {
	return base64Decode(s)
}
