package rankings

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
)

// NewPlayback builds the playback rankings command.
func NewPlayback(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var filterBy, period string
	var hasMagnets bool
	cmd := &cobra.Command{
		Use:   "playback",
		Short: "Playback rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			res, err := c.RankingsPlayback(context.Background(), filterBy, period)
			if err != nil {
				return fmt.Errorf("rankings failed: %w", err)
			}
			movies := res.Movies()
			if hasMagnets {
				movies = result.FilterMoviesWithMagnets(movies)
			}
			return writeMovies(streams.Out, streams.Err, movies)
		},
	}
	cmd.Flags().StringVar(&filterBy, "filter-by", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	return cmd
}
