// Package cache 提供本机反搜缓存的运维命令。
package cache

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/cache"
)

// New builds the cache commands.
func New(streams *environment.Streams) *cobra.Command {
	command := &cobra.Command{
		Use:   "cache",
		Short: i18n.T("command.cache"),
	}
	var source string
	var clear bool
	reverseSearchCommand := &cobra.Command{
		Use:   "reverse-search",
		Short: i18n.T("command.cache_reverse_search"),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := paths.ReverseSearchCacheDir()
			if err != nil {
				return err
			}
			store := cache.New(dir, 0)
			if clear {
				if source == "" {
					if err := store.Clear(""); err != nil {
						return err
					}
					fmt.Fprintln(streams.Out, i18n.T("cache.cleared"))
					return nil
				}
				if err := store.Clear(source); err != nil {
					return err
				}
				fmt.Fprintf(streams.Out, i18n.T("cache.source_cleared"), source)
				return nil
			}
			stats, err := store.Stats()
			if err != nil {
				return err
			}
			if source != "" {
				fmt.Fprintf(streams.Out, "%s=%d\n", source, stats[source])
				return nil
			}
			names := cache.SortedSources(stats)
			if len(names) == 0 {
				fmt.Fprintln(streams.Out, i18n.T("cache.empty"))
				return nil
			}
			for _, name := range names {
				fmt.Fprintf(streams.Out, "%s=%d\n", name, stats[name])
			}
			return nil
		},
	}
	reverseSearchCommand.Flags().StringVar(&source, "source", "", i18n.T("flag.cache_source"))
	reverseSearchCommand.Flags().BoolVar(&clear, "clear", false, i18n.T("flag.cache_clear"))
	command.AddCommand(reverseSearchCommand)
	return command
}
