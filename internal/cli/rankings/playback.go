package rankings

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/display"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// NewPlayback builds the playback rankings command.
func NewPlayback(options *session.Options, streams *environment.Streams) *cobra.Command {
	var filterBy, period string
	var hasMagnets bool
	var asJSON, asNDJSON bool
	producer := &record.ListProducer{
		Name: "rankings playback",
		ClientFactory: func() (*session.Client, error) {
			return session.NewWithDefaultToken(options)
		},
		Fetch: func(ctx context.Context, c *session.Client) ([]map[string]any, error) {
			res, err := c.RankingsPlayback(ctx, filterBy, period)
			if err != nil {
				return nil, err
			}
			movies := res.Movies()
			if hasMagnets {
				movies = display.FilterMoviesWithMagnets(movies)
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
		Short: i18n.T("cli.playback_rankings"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().StringVar(&filterBy, "filter-by", "censored", i18n.T("cli.censored_uncensored_western_fc"))
	cmd.Flags().StringVar(&period, "period", "day", i18n.T("cli.day_week_month"))
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, i18n.T("cli.drop_magnets_count"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.machine_readable_json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.pipeline_ndjson_envelopes"))
	return cmd
}

// writeMovies 用 movie 投影写出影片列表文本；空列表输出 (空列表)。
func writeMovies(w, errW io.Writer, movies []map[string]any) error {
	return record.WriteMovieRowsText(w, errW, movies)
}
