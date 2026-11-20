// Package i18n loads the embedded English and Simplified Chinese CLI catalogs.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

//go:embed en.json zh-CN.json
var files embed.FS

var catalogs = sync.OnceValues(func() (map[string]map[string]string, error) {
	out := make(map[string]map[string]string)
	for _, locale := range []string{"en", "zh-CN"} {
		raw, err := files.ReadFile(locale + ".json")
		if err != nil {
			return nil, err
		}
		var messages map[string]string
		if err := json.Unmarshal(raw, &messages); err != nil {
			return nil, fmt.Errorf("load %s messages: %w", locale, err)
		}
		out[locale] = messages
	}
	return out, nil
})

// Normalize maps system locale names to the supported catalogs.
func Normalize(locale string) string {
	locale = strings.ToLower(strings.ReplaceAll(locale, "_", "-"))
	if locale == "zh" || strings.HasPrefix(locale, "zh-") {
		return "zh-CN"
	}
	return "en"
}

// System honors POSIX locale overrides before the native user interface language.
func System() string {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if value := os.Getenv(key); value != "" {
			return Normalize(value)
		}
	}
	return Normalize(nativeLanguage())
}

// T reads a message using the current system language without changing global state.
func T(key string) string {
	all, err := catalogs()
	if err != nil {
		panic(err)
	}
	value, ok := all[System()][key]
	if !ok {
		panic("missing CLI message: " + key)
	}
	return value
}

// Errorf keeps Go's formatting and wrapped-error semantics for translated diagnostics.
func Errorf(key string, args ...any) error { return fmt.Errorf(T(key), args...) }
