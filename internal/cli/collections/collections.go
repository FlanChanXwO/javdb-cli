// Package collections 提供用户收藏列表命令。
package collections

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/display"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

var collectionKinds = map[string]record.Kind{
	"actors":    record.KindActor,
	"series":    record.KindSeries,
	"codes":     record.KindCode,
	"makers":    record.KindMaker,
	"directors": record.KindDirector,
}

func collectionKind(selector string) (record.Kind, error) {
	kind, ok := collectionKinds[selector]
	if !ok {
		return "", i18n.Errorf("collections.invalid_kind")
	}
	return kind, nil
}

func collectionEnvelope(selector string, kind record.Kind, item map[string]any) (record.Envelope, error) {
	row := display.ProjectNamed(item)
	if row.ID == "" {
		return record.Envelope{}, fmt.Errorf(i18n.T("collections.missing_id"), selector)
	}
	ref := row.Name
	if ref == "" {
		ref = row.ID
	}
	return record.New(kind, ref, row.ID).WithData(map[string]any{"entity": item}), nil
}

// New builds the collection listing command.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	var asJSON, asNDJSON bool
	runner := &record.BatchRunner{
		Name:       "collections",
		LegacyJSON: true,
		Preflight: func(inputs []record.Envelope) error {
			for _, input := range inputs {
				if _, err := collectionKind(record.ConsumerRef(input)); err != nil {
					return err
				}
			}
			return nil
		},
		ClientFactory: func() (*session.Client, error) {
			return session.NewRequired(options)
		},
		RunMany: func(c *session.Client, ctx context.Context, input record.Envelope) ([]record.Envelope, error) {
			kind := record.ConsumerRef(input)
			pipelineKind, err := collectionKind(kind)
			if err != nil {
				return nil, err
			}
			items, err := c.Collected(ctx, kind)
			if err != nil {
				return nil, err
			}
			envelopes := make([]record.Envelope, 0, len(items))
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
			return session.WithRequiredAuth(options, streams.Err, func(c *session.Client) error {
				items, err := c.Collected(context.Background(), kind)
				if err != nil {
					return err
				}
				if asJSON {
					return writeJSON(streams.Out, map[string]any{"items": items})
				}
				return writeNamed(streams.Out, streams.Diagnostics(slog.LevelInfo), items)
			})
		},
	}
	cmd := &cobra.Command{
		Use:   "collections KIND",
		Short: i18n.T("command.collections"),
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
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("flag.json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("flag.ndjson"))
	return cmd
}

// writeNamed 用 entity 投影写出命名实体列表文本；空列表输出 (空列表)。
func writeNamed(w, errW io.Writer, items []map[string]any) error {
	if len(items) == 0 {
		_, err := fmt.Fprintln(errW, i18n.T("result.empty"))
		return err
	}
	for _, row := range display.ProjectNamedAll(items) {
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
