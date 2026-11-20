package lists

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
	"github.com/FlanChanXwO/javdb-cli/sdk"
)

// NewSearch builds the lists search command.
func NewSearch(options *session.Options, streams *environment.Streams) *cobra.Command {
	var page, limit int
	var zone string
	var asJSON, asNDJSON bool
	runOne := func(c *session.Client, ctx context.Context, keyword string) ([]map[string]any, error) {
		res, err := c.Search(ctx, keyword, javdb.SearchOptions{
			Page: page, Limit: limit, Zone: zone, Type: "list",
		})
		if err != nil {
			return nil, err
		}
		return res.Named("lists"), nil
	}
	runner := &record.BatchRunner{
		Name:       "lists search",
		LegacyJSON: true,
		Kinds:      []record.Kind{record.KindList},
		ClientFactory: func() (*session.Client, error) {
			return session.NewWithDefaultToken(options)
		},
		RunMany: func(c *session.Client, ctx context.Context, input record.Envelope) ([]record.Envelope, error) {
			items, err := runOne(c, ctx, record.ConsumerRef(input))
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
			items, err := runOne(c, context.Background(), args[0])
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
		Use:   "search KEYWORD",
		Short: i18n.T("command.lists_search"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateZone(zone); err != nil {
				return err
			}
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().IntVar(&page, "page", 1, i18n.T("flag.page"))
	cmd.Flags().IntVar(&limit, "limit", 0, i18n.T("flag.limit"))
	cmd.Flags().StringVar(&zone, "zone", "all", "censored|uncensored|western|fc2|all")
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("flag.json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("flag.ndjson"))
	return cmd
}

func validateZone(zone string) error {
	switch zone {
	case "censored", "uncensored", "western", "fc2", "all":
		return nil
	default:
		return fmt.Errorf(i18n.T("search.invalid_zone"), zone)
	}
}
