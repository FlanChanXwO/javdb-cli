package rankings

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/cli/app"
	"github.com/FlanChanXwO/javdb-cli/internal/cli/entity"
)

// NewActors builds the actor rankings command.
func NewActors(flags *app.Flags, aio *app.IO) *cobra.Command {
	var period string
	cmd := &cobra.Command{
		Use:   "actors",
		Short: "Actor rankings",
		RunE: func(cmd *cobra.Command, args []string) error {
			rt, err := app.LoadRuntime(flags)
			if err != nil {
				return err
			}
			c, err := app.NewClient(rt, "")
			if err != nil {
				return err
			}
			res, err := c.RankingsActors(context.Background(), period)
			if err != nil {
				return fmt.Errorf("rankings failed: %w", err)
			}
			return writeNamedNoCount(aio.Out, aio.Err, res.Named("actors"))
		},
	}
	cmd.Flags().StringVar(&period, "period", "day", "day|week|month")
	return cmd
}

// writeNamedNoCount 写出 id\tname 行（忽略 count）；空列表输出 (空列表)。
func writeNamedNoCount(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range entity.ProjectNamedAll(items) {
		if _, err := fmt.Fprintf(w, "%s\t%s\n", row.ID, row.Name); err != nil {
			return err
		}
	}
	return nil
}
