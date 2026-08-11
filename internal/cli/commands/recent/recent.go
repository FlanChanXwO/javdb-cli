// Package recent 提供最近浏览影片列表命令。
package recent

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/movie"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the recently viewed movies command.
func New(flags *app.Flags, aio *app.IO) *cobra.Command {
	var hasMagnets bool
	cmd := &cobra.Command{
		Use:   "recent",
		Short: "List recently viewed (最近浏览) movies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.WithAuthedClient(flags, aio, func(c *javdb.Client) error {
				movies, err := c.RecentViewed(context.Background())
				if err != nil {
					return err
				}
				if hasMagnets {
					movies = movie.FilterHasMagnets(movies)
				}
				return writeMovies(aio.Out, aio.Err, movies)
			})
		},
	}
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
