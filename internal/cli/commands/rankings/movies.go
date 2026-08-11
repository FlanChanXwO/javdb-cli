package rankings

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
)

// NewMovies builds the movie rankings command.
func NewMovies(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var type_, period string
	var hasMagnets, asJSON bool
	cmd := &cobra.Command{
		Use:   "movies",
		Short: "Movie rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			res, err := c.RankingsMovies(context.Background(), type_, period)
			if err != nil {
				return fmt.Errorf("rankings failed: %w", err)
			}
			movies := res.Movies()
			if hasMagnets {
				movies = result.FilterMoviesWithMagnets(movies)
			}
			if asJSON {
				if movies == nil {
					movies = []map[string]any{}
				}
				return writeJSON(streams.Out, map[string]any{"movies": movies})
			}
			return writeMovies(streams.Out, streams.Err, movies)
		},
	}
	cmd.Flags().StringVar(&type_, "type", "censored", "censored|uncensored|western|fc2")
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, "Drop magnets_count==0")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
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

// writeJSON 以 jsonx.MarshalLine 写出紧凑 JSON 并传播编码与写入错误。
func writeJSON(w io.Writer, value any) error {
	b, err := jsonx.MarshalLine(value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}
