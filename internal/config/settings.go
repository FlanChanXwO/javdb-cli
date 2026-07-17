package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Host names accepted by --host / config.
const (
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
	Host         string `toml:"host"`
	HTTPSProxy   string `toml:"https_proxy,omitempty"`
	AutoRelogin  bool   `toml:"auto_relogin"`
	Lang         string `toml:"lang,omitempty"`
	DeviceUUID   string `toml:"device_uuid,omitempty"` // optional override; else file/device_uuid
}

// Defaults returns baseline settings.
func Defaults() Settings {
	return Settings{
		Host:        HostMirror,
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
		s.Host = HostMirror
	}
	return s, nil
}

// SaveFile writes settings sparsely to path (0600).
func SaveFile(path string, s Settings) error {
	if _, err := EnsureDir(); err != nil {
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
	Host        string // mirror | main
	BaseURL     string
	Proxy       string
	AutoRelogin bool
	Lang        string
	DeviceUUID  string
}

// Resolve merges flag/env/file/defaults.
// flagHost / flagProxy empty means "not set".
// flagAutoRelogin: nil = not set; non-nil overrides.
func Resolve(file Settings, flagHost, flagProxy string, flagAutoRelogin *bool) Runtime {
	r := Runtime{
		Host:        file.Host,
		Proxy:       file.HTTPSProxy,
		AutoRelogin: file.AutoRelogin,
		Lang:        file.Lang,
		DeviceUUID:  file.DeviceUUID,
	}
	if r.Host == "" {
		r.Host = HostMirror
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
		r.Proxy = flagProxy
	}
	if flagAutoRelogin != nil {
		r.AutoRelogin = *flagAutoRelogin
	}

	r.Host = strings.ToLower(strings.TrimSpace(r.Host))
	if url, ok := HostURLs[r.Host]; ok {
		r.BaseURL = url
	} else if strings.HasPrefix(r.Host, "http://") || strings.HasPrefix(r.Host, "https://") {
		r.BaseURL = strings.TrimRight(r.Host, "/")
	} else {
		// unknown name → treat as mirror but keep name for error messages
		r.BaseURL = HostURLs[HostMirror]
	}
	return r
}

// ValidateHost returns error if host is not a known name or absolute URL.
func ValidateHost(host string) error {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "" {
		return nil
	}
	if _, ok := HostURLs[h]; ok {
		return nil
	}
	if strings.HasPrefix(h, "http://") || strings.HasPrefix(h, "https://") {
		return nil
	}
	return fmt.Errorf("host must be mirror, main, or a URL, got %q", host)
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
