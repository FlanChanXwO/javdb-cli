package config

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/config/settings"
)

// New builds config path/get/set/unset commands.
func New(aio *app.IO) *cobra.Command {
	command := &cobra.Command{
		Use:   "config",
		Short: "Show or edit config.toml",
	}
	command.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print config file path",
		RunE: func(command *cobra.Command, args []string) error {
			path, err := paths.ConfigPath()
			if err != nil {
				return err
			}
			fmt.Fprintln(aio.Out, path)
			return nil
		},
	})
	command.AddCommand(&cobra.Command{
		Use:   "get [key]",
		Short: "Print config (or one key)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			path, err := paths.ConfigPath()
			if err != nil {
				return err
			}
			cfg, err := settings.LoadFile(path)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				fmt.Fprintf(aio.Out, "host=%s\nhttps_proxy=%s\nauto_relogin=%v\nlang=%s\n",
					cfg.Host, cfg.HTTPSProxy, cfg.AutoRelogin, cfg.Lang)
				return nil
			}
			switch args[0] {
			case "host":
				fmt.Fprintln(aio.Out, cfg.Host)
			case "https_proxy", "proxy":
				fmt.Fprintln(aio.Out, cfg.HTTPSProxy)
			case "auto_relogin":
				fmt.Fprintln(aio.Out, cfg.AutoRelogin)
			case "lang":
				fmt.Fprintln(aio.Out, cfg.Lang)
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
				if err := settings.ValidateHost(args[1]); err != nil {
					return err
				}
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
				cfg.Host = settings.HostMirror
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
