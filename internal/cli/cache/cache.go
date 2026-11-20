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
		Short: i18n.T("cli.inspect_or_clear_the_local_reverse_search_cache"),
	}
	var source string
	var clear bool
	reverseSearchCommand := &cobra.Command{
		Use:   "reverse-search",
		Short: i18n.T("cli.show_or_clear_the_reverse_search_response_cache"),
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
					fmt.Fprintln(streams.Out, i18n.T("cli.cleared_all_reverse_search_cache_entries"))
					return nil
				}
				if err := store.Clear(source); err != nil {
					return err
				}
				fmt.Fprintf(streams.Out, i18n.T("cli.cleared_reverse_search_cache_for_source_q"), source)
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
				fmt.Fprintln(streams.Out, i18n.T("cli.reverse_search_cache_is_empty"))
				return nil
			}
			for _, name := range names {
				fmt.Fprintf(streams.Out, "%s=%d\n", name, stats[name])
			}
			return nil
		},
	}
	reverseSearchCommand.Flags().StringVar(&source, "source", "", i18n.T("cli.limit_to_one_reverse_search_source"))
	reverseSearchCommand.Flags().BoolVar(&clear, "clear", false, i18n.T("cli.clear_cached_entries_instead_of_showing_stats"))
	command.AddCommand(reverseSearchCommand)
	return command
}
