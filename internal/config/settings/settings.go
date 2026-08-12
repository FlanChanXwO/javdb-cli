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
	Host        string `toml:"host"`
	HTTPSProxy  string `toml:"https_proxy,omitempty"`
	AutoRelogin bool   `toml:"auto_relogin"`
	Lang        string `toml:"lang,omitempty"`
	DeviceUUID  string `toml:"device_uuid,omitempty"` // optional override; else file/device_uuid
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
	return s, nil
}

// SaveFile writes settings sparsely to path (0600).
func SaveFile(path string, s Settings) error {
	if _, err := paths.EnsureDir(); err != nil {
		return err
	}
	data, err := toml.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
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
		Proxy:       file.HTTPSProxy,
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
	if v := firstEnv("HTTPS_PROXY", "https_proxy", "ALL_PROXY", "all_proxy"); v != "" {
		r.Proxy = v
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
	if flagProxy != "" {
		// 显式传入的空白 proxy 直接报错：当前文档只定义 --proxy URL，若裁剪成空串会静默
		// 覆盖继承代理并直连，绕过用户网络策略；config/env 来源的空白仍按空值规范。
		if strings.TrimSpace(flagProxy) == "" {
			return Runtime{}, fmt.Errorf("--proxy flag must be a non-empty URL, got %q", flagProxy)
		}
		r.Proxy = flagProxy
	}
	if flagAutoRelogin != nil {
		r.AutoRelogin = *flagAutoRelogin
	}

	// 空白 proxy（config/env/flag 任一来源）规范为空串，避免 validator 视为"空"但 transport
	// 仍收到原始空白而在请求阶段失败，且失败命令已留下本机状态。
	r.Proxy = strings.TrimSpace(r.Proxy)

	host, err := NormalizeHost(r.Host)
	if err != nil {
		return Runtime{}, err
	}
	r.Host = host
	if baseURL, ok := HostURLs[host]; ok {
		r.BaseURL = baseURL
	} else if host != HostAuto {
		r.BaseURL = host
	}
	return r, nil
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
