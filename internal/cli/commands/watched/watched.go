// Package watched 提供已看影片列表命令。
package watched

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the watched movies command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var hasMagnets bool
	var asJSON, asNDJSON bool
	producer := &pipeline.MovieListProducer{
		Name: "watched",
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		Fetch: func(ctx context.Context, c *javdb.Client) ([]map[string]any, error) {
			movies, err := c.WatchedMovies(ctx)
			if err != nil {
				return nil, err
			}
			if hasMagnets {
				movies = result.FilterMoviesWithMagnets(movies)
			}
			return movies, nil
		},
		JSON: func(movies []map[string]any) (map[string]any, error) {
			return map[string]any{"movies": movies}, nil
		},
	}
	cmd := &cobra.Command{
		Use:   "watched",
		Short: "List watched (看過) movies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
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
