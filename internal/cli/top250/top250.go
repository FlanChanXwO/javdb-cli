// Package top250 提供 TOP250 排行命令。
package top250

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/display"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the authenticated TOP250 command.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	var zone, year string
	var startRank, page, limit int
	var ignoreWatched, hasMagnets bool
	var asJSON, asNDJSON bool
	var generatedAt string
	producer := &record.MovieListProducer{
		Name: "top250",
		ClientFactory: func() (*session.Client, error) {
			return session.NewRequired(options)
		},
		Fetch: func(ctx context.Context, c *session.Client) ([]map[string]any, error) {
			res, err := c.Top250(ctx, zone, year, startRank, page, limit, ignoreWatched)
			if err != nil {
				return nil, err
			}
			generatedAt = string(jsonx.RawString(res["generated_at"]))
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
		RowText: writeRanked,
		ErrNote: func(w io.Writer, movies []map[string]any) {
			if generatedAt != "" {
				fmt.Fprintf(w, "# generated_at=%s\n", generatedAt)
			}
		},
	}
	cmd := &cobra.Command{
		Use:   "top250",
		Short: i18n.T("command.top250"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "", i18n.T("flag.ranking_zone"))
	cmd.Flags().StringVar(&year, "year", "", i18n.T("flag.ranking_year"))
	cmd.Flags().IntVar(&startRank, "from", 1, i18n.T("flag.start_rank"))
	cmd.Flags().IntVar(&page, "page", 1, i18n.T("flag.page"))
	cmd.Flags().IntVar(&limit, "limit", 20, i18n.T("flag.limit"))
	cmd.Flags().BoolVar(&ignoreWatched, "ignore-watched", false, i18n.T("flag.ignore_watched"))
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, i18n.T("flag.has_magnets"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("flag.json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("flag.ndjson"))
	return cmd
}

// writeRanked 写出带 #ranking 前缀的排行行；空列表输出 (空列表)。
func writeRanked(w, errW io.Writer, movies []map[string]any) error {
	if len(movies) == 0 {
		_, err := fmt.Fprintln(errW, i18n.T("result.empty"))
		return err
	}
	for _, m := range movies {
		row := display.ProjectMovie(m)
		rank := movieDisplay(m["ranking"])
		line := "#" + rank + "\t" + row.Line()
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func movieDisplay(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}
