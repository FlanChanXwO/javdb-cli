// Package top250 提供 TOP250 排行命令。
package top250

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the authenticated TOP250 command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var zone, year string
	var startRank, page, limit int
	var ignoreWatched, hasMagnets bool
	cmd := &cobra.Command{
		Use:   "top250",
		Short: "TOP250 list (needs login)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return client.WithRequiredAuth(options, streams.Err, func(c *javdb.Client) error {
				res, err := c.Top250(context.Background(), zone, year, startRank, page, limit, ignoreWatched)
				if err != nil {
					return fmt.Errorf("top250 failed: %w", err)
				}
				if gen := jsonx.RawString(res["generated_at"]); gen != "" {
					fmt.Fprintf(streams.Err, "# generated_at=%s\n", gen)
				}
				movies := res.Movies()
				if hasMagnets {
					movies = result.FilterMoviesWithMagnets(movies)
				}
				return writeRanked(streams.Out, streams.Err, movies)
			})
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "", "censored|uncensored|western|fc2 (omit for all-site)")
	cmd.Flags().StringVar(&year, "year", "", "Filter by year e.g. 2023")
	cmd.Flags().IntVar(&startRank, "from", 1, "Start from this rank")
	cmd.Flags().IntVar(&page, "page", 1, "Page")
	cmd.Flags().IntVar(&limit, "limit", 20, "Page size")
	cmd.Flags().BoolVar(&ignoreWatched, "ignore-watched", false, "Skip already watched titles")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
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
