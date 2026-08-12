// Package cache 提供本机反搜缓存的运维命令。
package cache

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/config/paths"
	"github.com/FlanChanXwO/javdb-cli/internal/reversesearch/cache"
)

// New builds the cache commands.
func New(streams *invocation.Streams) *cobra.Command {
	command := &cobra.Command{
		Use:   "cache",
		Short: "Inspect or clear the local reverse-search cache",
	}
	var source string
	var clear bool
	reverseSearchCommand := &cobra.Command{
		Use:   "reverse-search",
		Short: "Show or clear the reverse-search response cache",
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
					fmt.Fprintln(streams.Out, "cleared all reverse-search cache entries")
					return nil
				}
				if err := store.Clear(source); err != nil {
					return err
				}
				fmt.Fprintf(streams.Out, "cleared reverse-search cache for source %q\n", source)
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
				fmt.Fprintln(streams.Out, "reverse-search cache is empty")
				return nil
			}
			for _, name := range names {
				fmt.Fprintf(streams.Out, "%s=%d\n", name, stats[name])
			}
			return nil
		},
	}
	reverseSearchCommand.Flags().StringVar(&source, "source", "", "Limit to one reverse-search source")
	reverseSearchCommand.Flags().BoolVar(&clear, "clear", false, "Clear cached entries instead of showing stats")
	command.AddCommand(reverseSearchCommand)
	return command
}
