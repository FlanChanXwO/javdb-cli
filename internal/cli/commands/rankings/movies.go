package rankings

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewMovies builds the movie rankings command.
func NewMovies(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var type_, period string
	var hasMagnets bool
	var asJSON, asJSONL, asText bool
	producer := &pipeline.MovieListProducer{
		Name: "rankings movies",
		ClientFactory: func() (*javdb.Client, error) {
			return client.New(options, "")
		},
		Fetch: func(ctx context.Context, c *javdb.Client) ([]map[string]any, error) {
			res, err := c.RankingsMovies(ctx, type_, period)
			if err != nil {
				return nil, fmt.Errorf("rankings failed: %w", err)
			}
			movies := res.Movies()
			if hasMagnets {
				movies = result.FilterMoviesWithMagnets(movies)
			}
			return movies, nil
		},
		JSON: func(movies []map[string]any) (map[string]any, error) {
			if movies == nil {
				movies = []map[string]any{}
			}
			return map[string]any{"movies": movies}, nil
		},
	}
	cmd := &cobra.Command{
		Use:   "movies",
		Short: "Movie rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().StringVar(&type_, "type", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default for TTY)")
	return cmd
}

var _ = javdb.HostMain
