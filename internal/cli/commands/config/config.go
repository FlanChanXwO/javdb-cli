package config

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
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
	command.AddCommand(newGet(streams))
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
	command.AddCommand(newUnset(streams))
	return command
}

// newGet 构建 config get：一个 key 或无 key 时的非 TTY stdin 批处理。
func newGet(streams *invocation.Streams) *cobra.Command {
	var asJSON, asJSONL, asText bool
	load := func() (settings.Settings, string, error) {
		if err := paths.EnsureDefaultConfigFile(); err != nil {
			return settings.Settings{}, "", err
		}
		path, err := paths.ConfigPath()
		if err != nil {
			return settings.Settings{}, "", err
		}
		cfg, err := settings.LoadFile(path)
		if err != nil {
			return settings.Settings{}, "", err
		}
		return cfg, path, nil
	}
	runner := &pipeline.BatchRunner{
		Name:          "config get",
		Kinds:         []pipeline.Kind{pipeline.KindConfigKey},
		ClientFactory: nil,
		RunOne: func(_ *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			key := pipeline.ConsumerRef(input)
			if !knownConfigKey(key) {
				return pipeline.Envelope{}, fmt.Errorf("unknown key %q", key)
			}
			cfg, _, err := load()
			if err != nil {
				return pipeline.Envelope{}, err
			}
			value, err := lookupKey(cfg, key)
			if err != nil {
				return pipeline.Envelope{}, err
			}
			return pipeline.New(pipeline.KindConfigKey, key, "").
				WithData(map[string]any{"value": redactSensitiveValue(key, value)}), nil
		},
		Legacy: func(args []string) error {
			if !knownConfigKey(args[0]) {
				return fmt.Errorf("unknown key %q", args[0])
			}
			cfg, _, err := load()
			if err != nil {
				return err
			}
			value, err := lookupKey(cfg, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(streams.Out, value)
			return nil
		},
	}
	cmd := &cobra.Command{
		Use:   "get [key]",
		Short: "Print config (or one key)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			if len(args) == 1 && !knownConfigKey(args[0]) {
				return fmt.Errorf("unknown key %q", args[0])
			}
			if len(args) == 0 {
				// 无 key：TTY 打印全部；非 TTY 从 stdin 读取 key 批处理。
				cfg, _, err := load()
				if err != nil {
					return err
				}
				if streams.InIsTerminal {
					printAll(streams, cfg)
					return nil
				}
			}
			return runner.Execute(streams, args, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default for TTY)")
	return cmd
}

// newUnset 构建 config unset：一个 key 或非 TTY stdin key 批处理。
func newUnset(streams *invocation.Streams) *cobra.Command {
	var asJSON, asJSONL, asText bool
	runOne := func(key string) error {
		if !knownConfigKey(key) {
			return fmt.Errorf("unknown key %q", key)
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
		if err := document.Delete(key); err != nil {
			return err
		}
		return document.Save(path)
	}
	runner := &pipeline.BatchRunner{
		Name:          "config unset",
		Kinds:         []pipeline.Kind{pipeline.KindConfigKey},
		ClientFactory: nil,
		RunOne: func(_ *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			key := pipeline.ConsumerRef(input)
			if err := runOne(key); err != nil {
				return pipeline.Envelope{}, err
			}
			return pipeline.New(pipeline.KindConfigKey, key, "").WithData(map[string]any{"unset": true}), nil
		},
		Legacy: func(args []string) error {
			return runOne(args[0])
		},
	}
	cmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Clear a config key to default",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default for TTY)")
	return cmd
}

// redactSensitiveValue 对可能携带凭据的 proxy 值做脱敏：包含 userinfo 时
// 值以 "***" 替代，避免管道信封泄漏 secret。
func redactSensitiveValue(key, value string) string {
	if key != "https_proxy" && key != "proxy" {
		return value
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User == nil {
		return value
	}
	return "***"
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
