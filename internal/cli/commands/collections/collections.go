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

var collectionKinds = map[string]pipeline.Kind{
	"actors":    pipeline.KindActor,
	"series":    pipeline.KindSeries,
	"codes":     pipeline.KindCode,
	"makers":    pipeline.KindMaker,
	"directors": pipeline.KindDirector,
}

func collectionKind(selector string) (pipeline.Kind, error) {
	kind, ok := collectionKinds[selector]
	if !ok {
		return "", fmt.Errorf("collection kind must be one of actors|series|codes|makers|directors")
	}
	return kind, nil
}

func collectionEnvelope(selector string, kind pipeline.Kind, item map[string]any) (pipeline.Envelope, error) {
	row := result.ProjectNamed(item)
	if row.ID == "" {
		return pipeline.Envelope{}, fmt.Errorf("collections %s: entity has no id", selector)
	}
	ref := row.Name
	if ref == "" {
		ref = row.ID
	}
	return pipeline.New(kind, ref, row.ID).WithData(map[string]any{"entity": item}), nil
}

// New builds the collection listing command.
func New(options *invocation.RootOptions, streams *invocation.Streams) *cobra.Command {
	var asJSON, asNDJSON bool
	runner := &pipeline.BatchRunner{
		Name:       "collections",
		LegacyJSON: true,
		Preflight: func(inputs []pipeline.Envelope) error {
			for _, input := range inputs {
				if _, err := collectionKind(pipeline.ConsumerRef(input)); err != nil {
					return err
				}
			}
			return nil
		},
		ClientFactory: func() (*javdb.Client, error) {
			return client.NewWithDefaultToken(options)
		},
		RunMany: func(c *javdb.Client, ctx context.Context, input pipeline.Envelope) ([]pipeline.Envelope, error) {
			kind := pipeline.ConsumerRef(input)
			pipelineKind, err := collectionKind(kind)
			if err != nil {
				return nil, err
			}
			items, err := c.Collected(ctx, kind)
			if err != nil {
				return nil, err
			}
			envelopes := make([]pipeline.Envelope, 0, len(items))
			for _, item := range items {
				envelope, err := collectionEnvelope(kind, pipelineKind, item)
				if err != nil {
					return nil, err
				}
				envelopes = append(envelopes, envelope)
			}
			return envelopes, nil
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
			if len(args) == 1 {
				if _, err := collectionKind(args[0]); err != nil {
					return err
				}
			}
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Machine-readable JSON")
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, "Pipeline NDJSON envelopes")
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
