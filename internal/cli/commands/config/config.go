package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
)

// configKey 描述一个可 set/unset/get 的配置键及其值的 Go 类型。
type configKey struct {
	kind string // "string" | "bool" | "int"
}

// knownConfigKeys 是 config.toml 支持的键集合。未知键必须先报错，再决定是否
// 创建或读取配置，避免把无效命令的副作用落盘。
var knownConfigKeys = map[string]configKey{
	"host":                           {kind: "string"},
	"https_proxy":                    {kind: "string"},
	"proxy":                          {kind: "string"},
	"auto_relogin":                   {kind: "bool"},
	"lang":                           {kind: "string"},
	"reverse_search.default_source":  {kind: "string"},
	"reverse_search.cache":           {kind: "bool"},
	"reverse_search.cache_ttl":       {kind: "string"},
	"reverse_search.retries":         {kind: "int"},
	"reverse_search.retry_wait":      {kind: "string"},
	"reverse_search.request_timeout": {kind: "string"},
}

func knownConfigKey(key string) bool {
	_, ok := knownConfigKeys[key]
	return ok
}

func parseKeyValue(key, value string) (any, error) {
	switch knownConfigKeys[key].kind {
	case "bool":
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("key %q requires a boolean value, got %q", key, value)
		}
		return parsed, nil
	case "int":
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("key %q requires an integer value, got %q", key, value)
		}
		return parsed, nil
	default:
		return value, nil
	}
}

// displayConfigKeys 是 `config get` 无参输出顺序。
var displayConfigKeys = []string{
	"host",
	"https_proxy",
	"auto_relogin",
	"lang",
	"reverse_search.default_source",
	"reverse_search.cache",
	"reverse_search.cache_ttl",
	"reverse_search.retries",
	"reverse_search.retry_wait",
	"reverse_search.request_timeout",
}

// New builds config path/get/set/unset commands.
func New(streams *invocation.Streams) *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Show or edit config.toml",
	}
	command.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print config file path",
		RunE: func(command *cobra.Command, args []string) error {
			if err := paths.EnsureDefaultConfigFile(); err != nil {
				return err
			}
			path, err := paths.ConfigPath()
			if err != nil {
				return err
			}
			fmt.Fprintln(streams.Out, path)
			return nil
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "get [key]",
		Short: "Print config (or one key)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if len(args) == 1 && !knownConfigKey(args[0]) {
				return fmt.Errorf("unknown key %q", args[0])
			}
			if err := paths.EnsureDefaultConfigFile(); err != nil {
				return err
			}
			path, err := paths.ConfigPath()
			if err != nil {
				return err
			}
			cfg, err := settings.LoadFile(path)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				printAll(streams, cfg)
				return nil
			}
			value, err := lookupKey(cfg, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(streams.Out, value)
			return nil
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config key",
		Args:  cobra.ExactArgs(2),
		RunE: func(command *cobra.Command, args []string) error {
			if !knownConfigKey(args[0]) {
				return fmt.Errorf("unknown key %q", args[0])
			}
			// host 值语义校验必须先于创建，避免无效命令落盘基线配置。
			if args[0] == "host" {
				if err := settings.ValidateHost(args[1]); err != nil {
					return err
				}
			}
			value, err := parseKeyValue(args[0], args[1])
			if err != nil {
				return err
			}
			if err := paths.EnsureDefaultConfigFile(); err != nil {
				return err
			}
			path, err := paths.ConfigPath()
			if err != nil {
				return err
			}
			document, err := settings.LoadDocument(path)
			if err != nil {
				return err
			}
			if err := document.Set(args[0], value); err != nil {
				return err
			}
			return document.Save(path)
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "unset <key>",
		Short: "Clear a config key to default",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			// 未知 key 必须先报错（即使配置缺失），不能伪装成成功。
			if !knownConfigKey(args[0]) {
				return fmt.Errorf("unknown key %q", args[0])
			}
			path, err := paths.ConfigPath()
			if err != nil {
				return err
			}
			// 缺失配置上的合法 key unset 是 no-op：默认值已经生效，不创建文件。
			if _, err := os.Stat(path); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return nil
				}
				return err
			}
			document, err := settings.LoadDocument(path)
			if err != nil {
				return err
			}
			if err := document.Delete(args[0]); err != nil {
				return err
			}
			return document.Save(path)
		},
	})
	return command
}

func printAll(streams *invocation.Streams, cfg settings.Settings) {
	for _, key := range displayConfigKeys {
		value, err := lookupKey(cfg, key)
		if err != nil {
			continue
		}
		fmt.Fprintf(streams.Out, "%s=%s\n", key, value)
	}
}

func lookupKey(cfg settings.Settings, key string) (string, error) {
	switch key {
	case "host":
		return cfg.Host, nil
	case "https_proxy", "proxy":
		return cfg.HTTPSProxy, nil
	case "auto_relogin":
		return strconv.FormatBool(cfg.AutoRelogin), nil
	case "lang":
		return cfg.Lang, nil
	case "reverse_search.default_source":
		return cfg.ReverseSearch.DefaultSource, nil
	case "reverse_search.cache":
		return strconv.FormatBool(cfg.ReverseSearch.CacheEnabled()), nil
	case "reverse_search.cache_ttl":
		return cfg.ReverseSearch.CacheTTL, nil
	case "reverse_search.retries":
		return strconv.Itoa(cfg.ReverseSearch.Retries), nil
	case "reverse_search.retry_wait":
		return cfg.ReverseSearch.RetryWait, nil
	case "reverse_search.request_timeout":
		return cfg.ReverseSearch.RequestTimeout, nil
	default:
		return "", fmt.Errorf("unknown key %q", key)
	}
}
