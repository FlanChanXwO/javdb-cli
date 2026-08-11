// Package watched 提供已看影片列表命令。
package watched

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the watched movies command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var hasMagnets bool
	cmd := &cobra.Command{
		Use:   "watched",
		Short: "List watched (看過) movies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return client.WithRequiredAuth(options, streams.Err, func(c *javdb.Client) error {
				movies, err := c.WatchedMovies(context.Background())
				if err != nil {
					return err
				}
				if hasMagnets {
					movies = result.FilterMoviesWithMagnets(movies)
				}
				return writeMovies(streams.Out, streams.Err, movies)
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
	for _, row := range result.ProjectMovies(movies) {
		if _, err := fmt.Fprintln(w, row.Line()); err != nil {
			return err
		}
	}
	return nil
}
