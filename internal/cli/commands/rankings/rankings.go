// Package rankings 提供影片/演员/播放排行命令组。
package rankings

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
)

// New builds the movie, actor and playback rankings command group.
func New(flags *app.Flags, aio *app.IO) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rankings",
		Short: "Movie/actor rankings (no login needed)",
	}
	cmd.AddCommand(NewMovies(flags, aio))
	cmd.AddCommand(NewActors(flags, aio))
	cmd.AddCommand(NewPlayback(flags, aio))
	return cmd
}
