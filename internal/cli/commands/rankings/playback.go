package rankings

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/movie"
)

// NewPlayback builds the playback rankings command.
func NewPlayback(flags *app.Flags, aio *app.IO) *cobra.Command {
	var filterBy, period string
	var hasMagnets bool
	cmd := &cobra.Command{
		Use:   "playback",
		Short: "Playback rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := app.LoadRuntime(flags)
			if err != nil {
				return err
			}
			c, err := app.NewClient(rt, "")
			if err != nil {
				return err
			}
			res, err := c.RankingsPlayback(context.Background(), filterBy, period)
			if err != nil {
				return fmt.Errorf("rankings failed: %w", err)
			}
			movies := res.Movies()
			if hasMagnets {
				movies = movie.FilterHasMagnets(movies)
			}
			return writeMovies(aio.Out, aio.Err, movies)
		},
	}
	cmd.Flags().StringVar(&filterBy, "filter-by", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	return cmd
}
