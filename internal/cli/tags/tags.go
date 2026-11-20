// Package tags 提供内容标签 taxonomy 命令。
package tags

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/FlanChanXwO/javdb-cli/internal/common/jsonx"
	"github.com/FlanChanXwO/javdb-cli/internal/environment"
	"github.com/FlanChanXwO/javdb-cli/internal/i18n"
	"github.com/FlanChanXwO/javdb-cli/internal/record"
	"github.com/FlanChanXwO/javdb-cli/internal/session"
)

// New builds the taxonomy cache command.
func New(options *session.Options, streams *environment.Streams) *cobra.Command {
	var zone string
	var refresh bool
	var asJSON, asNDJSON bool
	producer := &record.Producer{
		Name: "tags",
		Produce: func(ctx context.Context) ([]record.Envelope, error) {
			c, err := session.NewWithDefaultToken(options)
			if err != nil {
				return nil, err
			}
			doc, path, err := c.LoadOrRefreshTaxonomy(ctx, zone, refresh)
			if err != nil {
				return nil, err
			}
			if refresh {
				fmt.Fprintf(streams.Diagnostics(slog.LevelInfo), i18n.T("cli.taxonomy_s"), path)
			}
			var envelopes []record.Envelope
			if doc == nil {
				return envelopes, nil
			}
			for _, cat := range doc.Categories {
				cname := cat.NameEN
				if cname == "" {
					cname = cat.NameZH
				}
				for _, t := range cat.Tags {
					envelopes = append(envelopes, record.New(record.KindTag, t.NameEN, t.ID).
						WithData(map[string]any{"category_id": cat.ID, "category": cname, "name_zh": t.NameZH}))
				}
			}
			return envelopes, nil
		},
		RenderText: func(w io.Writer, envelopes []record.Envelope) error {
			if len(envelopes) == 0 {
				_, err := fmt.Fprintln(streams.Diagnostics(slog.LevelInfo), i18n.T("cli.message_3"))
				return err
			}
			currentCategory := ""
			for _, envelope := range envelopes {
				category := fmt.Sprint(envelope.Data["category_id"])
				if category != currentCategory {
					currentCategory = category
					fmt.Fprintf(w, "# %s\t%s\n", category, envelope.Data["category"])
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", envelope.ID, envelope.Ref, envelope.Data["name_zh"])
			}
			return nil
		},
		LegacyJSON: func(w io.Writer) error {
			c, err := session.NewWithDefaultToken(options)
			if err != nil {
				return err
			}
			doc, _, err := c.LoadOrRefreshTaxonomy(context.Background(), zone, false)
			if err != nil {
				return err
			}
			var categories []map[string]any
			if doc != nil {
				for _, cat := range doc.Categories {
					var tagList []map[string]any
					for _, t := range cat.Tags {
						tagList = append(tagList, map[string]any{"id": t.ID, "name_en": t.NameEN, "name_zh": t.NameZH})
					}
					categories = append(categories, map[string]any{"id": cat.ID, "name_en": cat.NameEN, "name_zh": cat.NameZH, "tags": tagList})
				}
			}
			b, err := jsonx.MarshalLine(map[string]any{"categories": categories})
			if err != nil {
				return err
			}
			_, err = w.Write(b)
			return err
		},
	}
	cmd := &cobra.Command{
		Use:   "tags",
		Short: i18n.T("cli.list_content_tag_taxonomy_id_en"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return producer.Execute(streams, asNDJSON, asJSON)
		},
	}
	cmd.Flags().StringVar(&zone, "zone", "censored", i18n.T("cli.censored_uncensored_western_fc"))
	cmd.Flags().BoolVar(&refresh, "refresh", false, i18n.T("cli.re_fetch_from_api_and_rewrite_local_json"))
	cmd.Flags().BoolVar(&asJSON, "json", false, i18n.T("cli.machine_readable_json"))
	cmd.Flags().BoolVar(&asNDJSON, "ndjson", false, i18n.T("cli.pipeline_ndjson_envelopes"))
	return cmd
}
