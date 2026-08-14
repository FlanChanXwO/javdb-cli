package rankings

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewPlayback builds the playback rankings command.
func NewPlayback(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var filterBy, period string
	var hasMagnets bool
	var asJSON, asNDJSON bool
	producer := &pipeline.ListProducer{
		Name: "rankings playback",
		ClientFactory: func() (*javdb.Client, error) {
			return client.New(options, "")
		},
		Fetch: func(ctx context.Context, c *javdb.Client) ([]map[string]any, error) {
			res, err := c.RankingsPlayback(ctx, filterBy, period)
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
		Use:   "playback",
		Short: "Playback rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().StringVar(&filterBy, "filter-by", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
	return cmd
}

// writeMovies 用 movie 投影写出影片列表文本；空列表输出 (空列表)。
func writeMovies(w, errW io.Writer, movies []map[string]any) error {
	return pipeline.WriteMovieRowsText(w, errW, movies)
}
