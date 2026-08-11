package rankings

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/client"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/invocation"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/result"
)

// NewActors builds the actor rankings command.
func NewActors(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var period string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "actors",
		Short: "Actor rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := client.New(options, "")
			if err != nil {
				return err
			}
			res, err := c.RankingsActors(context.Background(), period)
			if err != nil {
				return fmt.Errorf("rankings failed: %w", err)
			}
			actors := res.Named("actors")
			if asJSON {
				if actors == nil {
					actors = []map[string]any{}
				}
				return writeJSON(streams.Out, map[string]any{"actors": actors})
			}
			return writeNamedNoCount(streams.Out, streams.Err, actors)
		},
	}
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
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
