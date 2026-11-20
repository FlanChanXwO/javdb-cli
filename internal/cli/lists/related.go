package lists

import (
	"context"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// NewRelated builds the lists related command.
func NewRelated(options *session.Options, streams *environment.Streams) *cobra.Command {
	var isID bool
	var page, limit int
	var asJSON, asNDJSON bool
	runOne := func(c *session.Client, ctx context.Context, ref string, useID bool) (string, []map[string]any, error) {
		mid := ref
		var err error
		if !useID {
			mid, err = c.ResolveMovieID(ctx, ref)
			if err != nil {
				return "", nil, err
			}
		}
		res, err := c.RelatedLists(ctx, mid, page, limit)
		if err != nil {
			return "", nil, err
		}
		return mid, res.Named("lists"), nil
	}
	runner := &record.BatchRunner{
		Name:       "lists related",
		LegacyJSON: true,
		Kinds:      []record.Kind{record.KindMovie},
		ClientFactory: func() (*session.Client, error) {
			return session.NewWithDefaultToken(options)
		},
		RunMany: func(c *session.Client, ctx context.Context, input record.Envelope) ([]record.Envelope, error) {
			ref := record.ConsumerRef(input)
			// 信封自带的 ID 是权威 movie ID；--id 只影响没有 ID 的输入。
			_, items, err := runOne(c, ctx, ref, input.ID != "" || isID)
			if err != nil {
				return nil, err
			}
			envelopes := make([]record.Envelope, 0, len(items))
			for _, item := range items {
				envelope, err := listEnvelope(item)
				if err != nil {
					return nil, err
				}
				envelopes = append(envelopes, envelope)
			}
			return envelopes, nil
		},
		Legacy: func(args []string) error {
			c, err := session.NewWithDefaultToken(options)
			if err != nil {
				return err
			}
			_, items, err := runOne(c, context.Background(), args[0], isID)
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(streams.Out, map[string]any{"lists": items})
			}
			return writeListRows(streams.Out, streams.Diagnostics(slog.LevelInfo), items)
		},
	}
	cmd := &cobra.Command{
		Use:   "related NUMBER",
		Short: i18n.T("command.lists_related"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVarP(&isID, "id", "i", false, i18n.T("flag.movie_id"))
	cmd.Flags().IntVar(&page, "page", 1, i18n.T("flag.page"))
	cmd.Flags().IntVar(&limit, "limit", 20, i18n.T("flag.limit"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("flag.json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("flag.ndjson"))
	return cmd
}
