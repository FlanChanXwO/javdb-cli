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

// NewActors builds the actor rankings command.
func NewActors(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var period string
	var asJSON, asNDJSON bool
	fetch := func(ctx context.Context, c *javdb.Client) ([]map[string]any, error) {
		res, err := c.RankingsActors(ctx, period)
		if err != nil {
			return nil, fmt.Errorf("rankings failed: %w", err)
		}
		actors := res.Named("actors")
		return actors, nil
	}
	producer := &pipeline.ListProducer{
		Name: "rankings actors",
		ClientFactory: func() (*javdb.Client, error) {
			return client.New(options, "")
		},
		Fetch: fetch,
		JSON: func(actors []map[string]any) (map[string]any, error) {
			if actors == nil {
				actors = []map[string]any{}
			}
			return map[string]any{"actors": actors}, nil
		},
		ItemKind: pipeline.KindActor,
		ItemRef: func(item map[string]any) (string, string) {
			return fmt.Sprint(item["name"]), fmt.Sprint(item["id"])
		},
		RowText: func(w, errW io.Writer, items []map[string]any) error {
			return writeNamedNoCount(w, errW, items)
		},
	}
	cmd := &cobra.Command{
		Use:   "actors",
		Short: "Actor rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
	return cmd
}

// writeNamedNoCount 写出 id\tname 行（忽略 count）；空列表输出 (空列表)。
func writeNamedNoCount(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range result.ProjectNamedAll(items) {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", row.ID, row.Name); err != nil {
			return err
		}
	}
	return nil
}
