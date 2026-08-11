package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
)

// knownConfigKey 报告 key 是否为 config.toml 支持的键。未知键必须先报错，再决定是否创建
// 或读取配置，避免把无效命令的副作用落盘。
func knownConfigKey(key string) bool {
	switch key {
	case "host", "https_proxy", "proxy", "auto_relogin", "lang":
		return true
	default:
		return false
	}
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
				fmt.Fprintf(streams.Out, "host=%s\nhttps_proxy=%s\nauto_relogin=%v\nlang=%s\n",
					cfg.Host, cfg.HTTPSProxy, cfg.AutoRelogin, cfg.Lang)
				return nil
			}
			switch args[0] {
			case "host":
				fmt.Fprintln(streams.Out, cfg.Host)
			case "https_proxy", "proxy":
				fmt.Fprintln(streams.Out, cfg.HTTPSProxy)
			case "auto_relogin":
				fmt.Fprintln(streams.Out, cfg.AutoRelogin)
			case "lang":
				fmt.Fprintln(streams.Out, cfg.Lang)
			default:
				return fmt.Errorf("unknown key %q", args[0])
			}
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
			switch args[0] {
			case "host":
				cfg.Host = args[1]
			case "https_proxy", "proxy":
				cfg.HTTPSProxy = args[1]
			case "auto_relogin":
				cfg.AutoRelogin = args[1] == "true" || args[1] == "1" || args[1] == "yes"
			case "lang":
				cfg.Lang = args[1]
			default:
				return fmt.Errorf("unknown key %q", args[0])
			}
			return settings.SaveFile(path, cfg)
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
			cfg, err := settings.LoadFile(path)
			if err != nil {
				return err
			}
			switch args[0] {
			case "host":
				cfg.Host = settings.HostAuto
			case "https_proxy", "proxy":
				cfg.HTTPSProxy = ""
			case "auto_relogin":
				cfg.AutoRelogin = false
			case "lang":
				cfg.Lang = "en"
			default:
				return fmt.Errorf("unknown key %q", args[0])
			}
			return settings.SaveFile(path, cfg)
		},
	})
	return command
}
