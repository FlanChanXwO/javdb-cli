// Package want 提供想看影片列表命令。
package want

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

// New builds the want-to-watch movies command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var hasMagnets bool
	var asJSON, asJSONL, asText bool
	producer := &pipeline.MovieListProducer{
		Name: "want",
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		Fetch: func(ctx context.Context, c *javdb.Client) ([]map[string]any, error) {
			movies, err := c.WantMovies(ctx)
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
		Use:   "want",
		Short: "List want-to-watch (想看) movies",
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default)")
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
