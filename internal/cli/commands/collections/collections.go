// Package collections 提供用户收藏列表命令。
package collections

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
	javdb "github.com/FlanChanXwO/javdb-cli/sdk"
)

// New builds the collection listing command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var asJSON, asJSONL, asText bool
	runner := &pipeline.BatchRunner{
		Name:       "collections",
		LegacyJSON: true,
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		RunOne: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) (pipeline.Envelope, error) {
			kind := pipeline.ConsumerRef(input)
			items, err := c.Collected(ctx, kind)
			if err != nil {
				return pipeline.Envelope{}, err
			}
			pipelineKind := pipeline.Kind(kind)
			return pipeline.New(pipelineKind, kind, "").WithData(map[string]any{"items": items}), nil
		},
		Legacy: func(args []string) error {
			kind := args[0]
			return client.WithRequiredAuth(options, streams.Err, func(c *javdb.Client) error {
				items, err := c.Collected(context.Background(), kind)
				if err != nil {
					return err
				}
				if asJSON {
					return writeJSON(streams.Out, map[string]any{"items": items})
				}
				return writeNamed(streams.Out, streams.Err, items)
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "collections KIND",
		Short: "List a collection: actors|series|codes|makers|directors",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asJSONL, asText, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asJSONL, "jsonl", false, "Pipeline JSONL envelopes")
	cmd.Flags().BoolVar(&asText, "text", false, "Plain text lines (default for TTY)")
	return cmd
}

// writeNamed 用 entity 投影写出命名实体列表文本；空列表输出 (空列表)。
func writeNamed(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := errW.Write([]byte("(空列表)\n"))
		return err
	}
	for _, row := range result.ProjectNamedAll(items) {
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
