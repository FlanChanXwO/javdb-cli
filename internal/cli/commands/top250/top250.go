// Package top250 提供 TOP250 排行命令。
package top250

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/pipeline"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the authenticated TOP250 command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var zone, year string
	var startRank, page, limit int
	var ignoreWatched, hasMagnets bool
	var asJSON, asJSONL, asText bool
	var generatedAt string
	producer := &pipeline.MovieListProducer{
		Name: "top250",
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		Fetch: func(ctx context.Context, c *javdb.Client) ([]map[string]any, error) {
			res, err := c.Top250(ctx, zone, year, startRank, page, limit, ignoreWatched)
			if err != nil {
				return nil, fmt.Errorf("top250 failed: %w", err)
			}
			generatedAt = string(jsonx.RawString(res["generated_at"]))
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
		RowText: writeRanked,
		ErrNote: func(w io.Writer, movies []map[string]any) {
			if generatedAt != "" {
				fmt.Fprintf(w, "# generated_at=%s\n", generatedAt)
			}
		},
	}
	cmd := &cobra.Command{
		Use:   "top250",
		Short: "TOP250 list (needs login)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "", "censored|uncensored|western|fc2 (omit for all-site)")
	cmd.Flags().StringVar(&year, "year", "", "Filter by year e.g. 2023")
	cmd.Flags().IntVar(&startRank, "from", 1, "Start from this rank")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&ignoreWatched, "ignore-watched", false, "Skip already watched titles")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default for TTY)")
	return cmd
}

// writeRanked 写出带 #ranking 前缀的排行行；空列表输出 (空列表)。
func writeRanked(w, errW io.Writer, movies []map[string]any) error {
	if len(movies) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, m := range movies {
		row := result.ProjectMovie(m)
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
