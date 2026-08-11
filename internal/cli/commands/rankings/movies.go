package rankings

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/movie"
)

// NewMovies builds the movie rankings command.
func NewMovies(flags *app.Flags, aio *app.IO) *cobra.Command {
	var type_, period string
	var hasMagnets bool
	cmd := &cobra.Command{
		Use:   "movies",
		Short: "Movie rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := app.LoadRuntime(flags)
			if err != nil {
				return err
			}
			c, err := app.NewClient(rt, "")
			if err != nil {
				return err
			}
			res, err := c.RankingsMovies(context.Background(), type_, period)
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
	cmd.Flags().StringVar(&type_, "type", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	return cmd
}

func writeMovies(w, errW io.Writer, movies []map[string]any) error {
	if len(movies) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range movie.ProjectAll(movies) {
		if _, err := fmt.Fprintln(w, row.Line()); err != nil {
			return err
		}
	}
	return nil
}
