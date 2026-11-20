package lists

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/common/scalar"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// NewShow builds the lists show command.
func NewShow(options *session.Options, streams *environment.Streams) *cobra.Command {
	var asJSON, asNDJSON bool
	runOne := func(c *session.Client, ctx context.Context, ref string) (string, map[string]any, error) {
		eid, err := c.ResolveEntity(ctx, "list", ref, "censored")
		if err != nil {
			return "", nil, err
		}
		data, err := c.ListInfo(ctx, eid)
		if err != nil {
			return "", nil, err
		}
		return eid, data, nil
	}
	runner := &record.BatchRunner{
		Name:       "lists show",
		LegacyJSON: true,
		Kinds:      []record.Kind{record.KindList},
		ClientFactory: func() (*session.Client, error) {
			return session.NewWithDefaultToken(options)
		},
		RunOne: func(c *session.Client, ctx context.Context, input record.Envelope) (record.Envelope, error) {
			eid, data, err := runOne(c, ctx, record.ConsumerRef(input))
			if err != nil {
				return record.Envelope{}, err
			}
			return record.New(record.KindList, input.Ref, eid).WithData(map[string]any{"list_info": data}), nil
		},
		Legacy: func(args []string) error {
			c, err := session.NewWithDefaultToken(options)
			if err != nil {
				return err
			}
			ctx := context.Background()
			eid, data, err := runOne(c, ctx, args[0])
			if err != nil {
				return err
			}
			if asJSON {
				return writeJSON(streams.Out, data)
			}
			_ = eid
			meta, _ := data["list"].(map[string]any)
			if meta == nil {
				if m, ok := data["list"]; ok {
					b, _ := json.Marshal(m)
					_ = json.Unmarshal(b, &meta)
				}
			}
			if meta == nil {
				meta = data
			}
			fmt.Fprintf(streams.Out, i18n.T("cli.id_s"), coalesce(display(meta["id"]), eid))
			fmt.Fprintf(streams.Out, i18n.T("cli.name_s"), display(meta["name"]))
			if d := display(meta["description"]); d != "" {
				fmt.Fprintf(streams.Out, i18n.T("cli.desc_s"), d)
			}
			fmt.Fprintf(streams.Out, i18n.T("cli.movies_s"), display(meta["movies_count"]))
			fmt.Fprintf(streams.Out, i18n.T("cli.views_s"), display(meta["views_count"]))
			fmt.Fprintf(streams.Out, i18n.T("cli.collects_s"), display(meta["collections_count"]))
			if s := display(meta["share_info"]); s != "" {
				fmt.Fprintf(streams.Out, i18n.T("cli.share_s"), s)
			}
			fmt.Fprintf(streams.Out, i18n.T("cli.is_creator_v"), data["is_creator"])
			fmt.Fprintf(streams.Out, i18n.T("cli.has_collected_v"), data["has_collected"])
			return nil
		},
	}
	cmd := &cobra.Command{
		Use:   "show REF",
		Short: i18n.T("cli.show_meta_movies_use_list_id"),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runner.Execute(streams, args, asNDJSON, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.json_output"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.pipeline_ndjson_envelopes"))
	return cmd
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// display 保持 CLI 既有的数值 ID 截断展示约定。
func display(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatInt(int64(t), 10)
	default:
		return scalar.String(t)
	}
}
