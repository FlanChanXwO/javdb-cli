// Package rankings 提供影片/演员/播放排行命令组。
package rankings

import (
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/spf13/cobra"
)

// New builds the movie, actor and playback rankings command group.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rankings",
		Short: "Movie/actor rankings (no login needed)",
	}
	cmd.AddCommand(NewMovies(options, streams))
	cmd.AddCommand(NewActors(options, streams))
	cmd.AddCommand(NewPlayback(options, streams))
	return cmd
}
