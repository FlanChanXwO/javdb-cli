package rankings

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

// NewActors builds the actor rankings command.
func NewActors(options *session.Options, streams *environment.Streams) *cobra.Command {
	var period string
	var asJSON, asNDJSON bool
	fetch := func(ctx context.Context, c *session.Client) ([]map[string]any, error) {
		res, err := c.RankingsActors(ctx, period)
		if err != nil {
			return nil, err
		}
		actors := res.Named("actors")
		return actors, nil
	}
	producer := &record.ListProducer{
		Name: "rankings actors",
		ClientFactory: func() (*session.Client, error) {
			return session.NewWithDefaultToken(options)
		},
		Fetch: fetch,
		JSON: func(actors []map[string]any) (map[string]any, error) {
			if actors == nil {
				actors = []map[string]any{}
			}
			return map[string]any{"actors": actors}, nil
		},
		ItemKind: record.KindActor,
		ItemRef: func(item map[string]any) (string, string) {
			return fmt.Sprint(item["name"]), fmt.Sprint(item["id"])
		},
		RowText: func(w, errW io.Writer, items []map[string]any) error {
			return writeNamedNoCount(w, errW, items)
		},
	}
	cmd := &cobra.Command{
		Use:   "actors",
		Short: i18n.T("cli.actor_rankings"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().StringVar(&period, "period", "day", i18n.T("cli.day_week_month"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.machine_readable_json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.pipeline_ndjson_envelopes"))
	return cmd
}

// writeNamedNoCount 写出 id\tname 行（忽略 count）；空列表输出 (空列表)。
func writeNamedNoCount(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := errW.Write([]byte(i18n.T("cli.empty_list_line")))
		return err
	}
	for _, row := range display.ProjectNamedAll(items) {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", row.ID, row.Name); err != nil {
			return err
		}
	}
	return nil
}
