// Package settings 管理 config.toml 的 schema 与优先级解析。
package settings

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/pelletier/go-toml/v2"
)

// Host names accepted by --host / config.
const (
	HostAuto   = "auto"
	HostMirror = "mirror"
	HostMain   = "main"
)

// HostURLs maps logical host names to base URLs.
var HostURLs = map[string]string{
	HostMirror: "https://jdforrepam.com",
	HostMain:   "https://javdb.com",
}

// Settings is the on-disk config.toml schema.
type Settings struct {
	Host          string                `toml:"host"`
	HTTPSProxy    string                `toml:"https_proxy,omitempty"`
	AutoRelogin   bool                  `toml:"auto_relogin"`
	Lang          string                `toml:"lang,omitempty"`
	DeviceUUID    string                `toml:"device_uuid,omitempty"` // optional override; else file/device_uuid
	ReverseSearch ReverseSearchSettings `toml:"reverse_search"`
}

// Defaults returns baseline settings.
func Defaults() Settings {
	return Settings{
		Host:        HostAuto,
		AutoRelogin: false,
		Lang:        "en",
	}
}

// LoadFile reads config.toml; missing file returns Defaults().
func LoadFile(path string) (Settings, error) {
	s := Defaults()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	if err := toml.Unmarshal(data, &s); err != nil {
		return s, err
	}
	if s.Host == "" {
		s.Host = HostAuto
	}
	s.ReverseSearch.applyDefaults()
	return s, nil
}

// SaveFile writes settings sparsely to path (0600). 保存采用结构化合并：先读
// 既有配置树，只覆盖已知顶层键，未修改的表、source 数组与未知键保持原样。
func SaveFile(path string, s Settings) error {
	if _, err := paths.EnsureDir(); err != nil {
		return err
	}
	document, err := LoadDocument(path)
	if err != nil {
		return err
	}
	if err := document.Set("host", s.Host); err != nil {
		return err
	}
	if s.HTTPSProxy != "" {
		if err := document.Set("https_proxy", s.HTTPSProxy); err != nil {
			return err
		}
	} else if err := document.Delete("https_proxy"); err != nil {
		return err
	}
	if err := document.Set("auto_relogin", s.AutoRelogin); err != nil {
		return err
	}
	if s.Lang != "" {
		if err := document.Set("lang", s.Lang); err != nil {
			return err
		}
	}
	if s.DeviceUUID != "" {
		if err := document.Set("device_uuid", s.DeviceUUID); err != nil {
			return err
		}
	}
	return document.Save(path)
}

// Runtime is the resolved config after flag > env > file > default.
type Runtime struct {
	Host        string // auto | mirror | main | URL
	BaseURL     string
	Proxy       string
	AutoRelogin bool
	Lang        string
	DeviceUUID  string
}

// Resolve merges flag/env/file/defaults.
// flagHost / flagProxy empty means "not set".
// flagAutoRelogin: nil = not set; non-nil overrides.
func Resolve(file Settings, flagHost, flagProxy string, flagAutoRelogin *bool) (Runtime, error) {
	r := Runtime{
		Host:        file.Host,
		AutoRelogin: file.AutoRelogin,
		Lang:        file.Lang,
		DeviceUUID:  file.DeviceUUID,
	}
	if r.Host == "" {
		r.Host = HostAuto
	}
	if r.Lang == "" {
		r.Lang = "en"
	}

	// env
	if v := firstEnv("JAVDB_HOST"); v != "" {
		r.Host = v
	}
	if v := firstEnv("JAVDB_AUTO_RELOGIN"); v != "" {
		r.AutoRelogin = parseBool(v)
	}
	if v := firstEnv("JAVDB_LANG"); v != "" {
		r.Lang = v
	}

	// flags win
	if flagHost != "" {
		r.Host = flagHost
	}
	if flagAutoRelogin != nil {
		r.AutoRelogin = *flagAutoRelogin
	}

	proxy, err := ResolveProxy(file, flagProxy)
	if err != nil {
		return Runtime{}, err
	}
	r.Proxy = proxy

	host, err := NormalizeHost(r.Host)
	if err != nil {
		return Runtime{}, err
	}
	if host == "" {
		host = HostAuto
	}
	r.Host = host
	if baseURL, ok := HostURLs[host]; ok {
		r.BaseURL = baseURL
	} else if host != HostAuto {
		r.BaseURL = host
	}
	return r, nil
}

// ResolveProxy 按 flag > env > file 的优先级解析代理，供无需 JavDB host 的独立流程复用。
func ResolveProxy(file Settings, flagProxy string) (string, error) {
	proxy := file.HTTPSProxy
	if v := firstEnv("HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy"); v != "" {
		proxy = v
	}
	if flagProxy != "" {
		// 显式空白会静默覆盖继承代理并直连，因此与“未设置”区分并直接拒绝。
		if strings.TrimSpace(flagProxy) == "" {
			return "", fmt.Errorf("--proxy flag must be a non-empty URL, got %q", flagProxy)
		}
		proxy = flagProxy
	}
	return strings.TrimSpace(proxy), nil
}

// NormalizeHost 校验并规范逻辑 host 或绝对 HTTP(S) URL。绝对 URL 必须不含 query 或
// fragment，否则 transport 以字符串拼接追加 API path 时会把 endpoint 拼进 query 或根本不
// 发给服务器。
func NormalizeHost(host string) (string, error) {
	h := strings.TrimSpace(host)
	if h == "" {
		return "", nil
	}
	logical := strings.ToLower(h)
	if logical == HostAuto {
		return HostAuto, nil
	}
	if _, ok := HostURLs[logical]; ok {
		return logical, nil
	}
	u, err := url.Parse(h)
	if err == nil && u.IsAbs() && u.Host != "" &&
		(strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")) &&
		!strings.Contains(h, "?") && !strings.Contains(h, "#") {
		return strings.TrimRight(h, "/"), nil
	}
	return "", fmt.Errorf("host must be auto, mirror, main, or an absolute HTTP(S) URL, got %q", host)
}

// ValidateHost returns error if host is not a known name or absolute URL.
func ValidateHost(host string) error {
	_, err := NormalizeHost(host)
	return err
}

func firstEnv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func parseBool(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		// treat "1"/"yes"/"on" loosely
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "1", "yes", "on", "true":
			return true
		default:
			return false
		}
	}
	return b
}
