// Package watched 提供已看影片列表命令。
package watched

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/display"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the watched movies command.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	var hasMagnets bool
	var asJSON, asNDJSON bool
	producer := &record.MovieListProducer{
		Name: "watched",
		ClientFactory: func() (*session.Client, error) {
			return session.NewRequired(options)
		},
		Fetch: func(ctx context.Context, c *session.Client) ([]map[string]any, error) {
			movies, err := c.WatchedMovies(ctx)
			if err != nil {
				return nil, err
			}
			if hasMagnets {
				movies = display.FilterMoviesWithMagnets(movies)
			}
			return movies, nil
		},
		JSON: func(movies []map[string]any) (map[string]any, error) {
			return map[string]any{"movies": movies}, nil
		},
	}
	cmd := &cobra.Command{
		Use:   "watched",
		Short: i18n.T("command.watched"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVar(&hasMagnets, "has-magnets", false, i18n.T("flag.has_magnets"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("flag.json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("flag.ndjson"))
	return cmd
}

func writeMovies(w, errW io.Writer, movies []map[string]any) error {
	if len(movies) == 0 {
		_, err := fmt.Fprintln(errW, i18n.T("result.empty"))
		return err
	}
	for _, row := range display.ProjectMovies(movies) {
		if _, err := fmt.Fprintln(w, row.Line()); err != nil {
			return err
		}
	}
	return nil
}
