// Package rankings 提供影片/演员/播放排行命令组。
package rankings

import (
	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the movie, actor and playback rankings command group.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rankings",
		Short: i18n.T("cli.movie_actor_rankings_no_login_needed"),
	}
	cmd.AddCommand(NewMovies(options, streams))
	cmd.AddCommand(NewActors(options, streams))
	cmd.AddCommand(NewPlayback(options, streams))
	return cmd
}
